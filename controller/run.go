package controller

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"fmt"
	"net/url"
	"os"
	path_helper "path"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/BurntSushi/toml"
	"github.com/adrg/xdg"
	"github.com/briandowns/spinner"
	"github.com/charmbracelet/huh"
	"github.com/charmbracelet/lipgloss"
	"github.com/hashicorp/go-getter"
	"github.com/hashicorp/go-hclog"
	"github.com/julien040/anyquery/controller/config/model"
	"github.com/julien040/anyquery/controller/config/registry"
	"github.com/julien040/anyquery/namespace"
	"github.com/julien040/anyquery/other/sqlparser"
	"github.com/spf13/cobra"
	"golang.org/x/term"
)

var regexpManifest = regexp.MustCompile(`/\*([^*]|\*+[^*/])*\*+/`)

type manifestQuery struct {
	Title       string `toml:"title"`
	Description string `toml:"description"`
	Author      string `toml:"author"`
	Args        []struct {
		Title        string `toml:"title"`
		DisplayTitle string `toml:"display_title"`
		Description  string `toml:"description"`
		Type         string `toml:"type"`
		Regex        string `toml:"regex"`
	} `toml:"arguments"`
	Plugins []string
}

func Run(cmd *cobra.Command, args []string) error {
	path := "anyquery.db"
	var inMemory, readOnly bool

	// Get the flags
	path, _ = cmd.Flags().GetString("database")
	inMemory, _ = cmd.Flags().GetBool("in-memory")
	readOnly, _ = cmd.Flags().GetBool("readonly")
	if !readOnly {
		readOnly, _ = cmd.Flags().GetBool("read-only")
	}

	if path == ":memory:" {
		inMemory = true
	}

	// If the path is empty, we open an in-memory database
	if path == "" {
		inMemory = true
	}

	if inMemory {
		// Does not matter so we set it to a random value
		// We set it a random value because if the database is in memory,
		// the first arg will be the query, and not the database path
		path = "myrandom.db"
	}

	// Check for the query ID or URL
	queryID := ""
	if len(args) > 0 {
		queryID = args[0]
	}
	if queryID == "" {
		return fmt.Errorf("query ID or URL is required")
	}

	// Download the query
	// We have to check if the query is a URL or an ID
	isID := false
	parsed, err := url.Parse(queryID)

	switch {
	case err == nil && parsed.Scheme != "" && parsed.Host != "":
		// The query is an URL
		isID = false
	case err == nil && parsed.Scheme == "" && parsed.Host == "":
		// The query might be an ID or a local path
		// We have to check if the query is a local path
		if _, err := os.Stat(queryID); err == nil {
			// The query is a local path
			isID = false
		} else {
			// The query is an ID
			isID = true
		}
	default:
		return fmt.Errorf("invalid query ID or URL")

	}

	urlToFetch := ""
	if isID {
		urlToFetch = fmt.Sprintf("https://registry.anyquery.dev/v0/query/%s", url.PathEscape(queryID))
	} else {
		urlToFetch = queryID
	}

	// Download the query
	hashedURL := fmt.Sprintf("%x", sha256.Sum256([]byte(urlToFetch)))
	dest := path_helper.Join(xdg.CacheHome, "anyquery", "queries", hashedURL)
	err = os.MkdirAll(path_helper.Dir(dest), 0755)
	if err != nil {
		return fmt.Errorf("failed to create the directory for the query: %s", err)
	}
	wd, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("failed to get the current working directory: %s", err)
	}

	// Download the file if needed (or older than 1 day)
	secondsInDay := int64(60 * 60 * 24)
	if fileInfo, err := os.Stat(dest); err != nil || fileInfo.Size() == 0 || time.Now().Unix()-fileInfo.ModTime().Unix() > secondsInDay {
		client := &getter.Client{
			Src:  urlToFetch,
			Dst:  dest,
			Mode: getter.ClientModeFile,
			Pwd:  wd,
		}

		err = client.Get()
		if err != nil {
			return fmt.Errorf("failed to download the query: %s", err)
		}
	}

	// Open the file
	content, err := os.ReadFile(dest)
	if err != nil {
		return fmt.Errorf("failed to read the query: %s", err)
	}

	// The file has the following content
	/*
		/*
		title = "GitHub stars per day"
		description = "This query returns the number of stars per day for a GitHub repository."
		author = "Albert Einstein"

		args = [
			{title = "Repository", description = "The GitHub repository to query", type = "string", regex = "^[a-zA-Z0-9_-]+/[a-zA-Z0-9_-]+$"},
		]

		plugins = [ "github" ]

		* /

		SELECT
			date(starred_at) as day,
			count(*)
		FROM github_stargazers_from_repository($1)
		GROUP BY day
		ORDER BY day;

	*/
	// We have to parse the file to extract the top-level comments using a regex
	// Then, we parse it so that we can extract the query and the arguments
	// We can then run the query

	// Extract the manifestStr
	manifestStr := regexpManifest.FindString(string(content))
	if manifestStr == "" {
		return fmt.Errorf("invalid query: missing manifest")
	}
	manifestStr = strings.TrimPrefix(manifestStr, "/*")
	manifestStr = strings.TrimSuffix(manifestStr, "*/")

	// Extract the query
	manifest := manifestQuery{}
	err = toml.Unmarshal([]byte(manifestStr), &manifest)
	if err != nil {
		return fmt.Errorf("failed to parse the TOML manifest: %s", err)
	}
	if manifest.Title == "" {
		return fmt.Errorf("invalid manifest: missing title")
	}

	// Let's open the config database
	// so that we can check which plugins need to be installed
	db, queries, err := requestDatabase(cmd.Flags(), false)
	if err != nil {
		return fmt.Errorf("could not open the database: %w", err)
	}
	defer db.Close()

	// Update the registries if needed
	s := spinner.New(spinner.CharSets[11], 100*time.Millisecond)
	s.Prefix = "Updating registries "
	if isSTDoutAtty() {
		s.Start()
	}
	// Ensure the database is up to date with the registries
	err = checkRegistries(queries)
	s.Stop() // no-op if not started
	if err != nil {
		return fmt.Errorf("could not update the registries: %w", err)
	}

	// Check if the plugins are installed
	for _, plugin := range manifest.Plugins {
		// Sometimes, plugin can be downloaded from a different reg
		// We therefore have to check if the plugin is in the reg
		reg := "default"
		splitted := strings.Split(plugin, "/")
		var pluginNeedsToBeInstalled, profileNeedsToBeCreated bool
		if len(splitted) >= 2 {
			reg = splitted[0]
			plugin = splitted[1]
		}
		_, err := queries.GetPlugin(context.Background(), model.GetPluginParams{
			Name:     plugin,
			Registry: reg,
		})
		pluginNeedsToBeInstalled = err != nil

		// Check if the profile needs to be created
		profiles, err := queries.GetProfilesOfPlugin(context.Background(), model.GetProfilesOfPluginParams{
			Pluginname: plugin,
			Registry:   reg,
		})

		profileNeedsToBeCreated = err != nil || len(profiles) == 0

		// Download the plugin if needed
		if pluginNeedsToBeInstalled {
			s.Prefix = "Installing plugin " + plugin + " "
			if isSTDoutAtty() {
				s.Start()
			}
			_, err = registry.InstallPlugin(queries, reg, plugin)
			s.Stop() // no-op if not started
			if err != nil {
				return fmt.Errorf("could not install the plugin: %w", err)
			}

			fmt.Println("✅ Successfully installed the plugin", plugin)
		}

		// Create a profile if needed
		if profileNeedsToBeCreated {
			err = createOrUpdateProfile(queries, reg, plugin, "default")
			if err != nil {
				return fmt.Errorf("could not create the profile: %w", err)
			}
		}
	}

	// Create the namespace so that we can run the query
	namespace, err := namespace.NewNamespace(namespace.NamespaceConfig{
		InMemory: inMemory,
		ReadOnly: readOnly,
		Path:     path,
		Logger:   hclog.NewNullLogger(),
	})

	if err != nil {
		return fmt.Errorf("failed to create namespace: %w", err)
	}

	anyqueryConfigPath := ""
	anyqueryConfigPath, err = cmd.Flags().GetString("config")
	if anyqueryConfigPath == "" {
		anyqueryConfigPath, err = xdg.ConfigFile("anyquery/config.db")
		if err != nil {
			return err
		}
	}

	err = namespace.LoadAsAnyqueryCLI(anyqueryConfigPath)
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	// Register the namespace
	db, err = namespace.Register("main")
	if err != nil {
		return fmt.Errorf("failed to register namespace: %w", err)
	}

	renderer := lipgloss.NewRenderer(os.Stdout)
	// Purple background, padding top, left, right 2, bottom 1
	titleStyle := renderer.NewStyle().Bold(true).Width(60).Foreground(lipgloss.Color("#f1f1f1")).Background(lipgloss.Color("#6f42c1")).Padding(1, 2, 0, 2).MarginTop(1)
	fmt.Fprintln(os.Stdout, titleStyle.Render(manifest.Title))
	// Print the description (purple background, padding 0 top, 2 left, 2 right, 2 bottom)
	descriptionStyle := renderer.NewStyle().Width(60).Foreground(lipgloss.Color("#B4B4B4")).Background(lipgloss.Color("#6f42c1")).Padding(0, 2, 1, 2).MarginBottom(1)
	fmt.Fprintln(os.Stdout, descriptionStyle.Render(manifest.Description))

	// Request the arguments
	fields := make([]huh.Field, 0, len(manifest.Args))
	rawAnswers := make([]string, len(manifest.Args))
	for i, arg := range manifest.Args {
		title := arg.Title
		// Make sure the title is not empty
		if title == "" {
			return fmt.Errorf("argument title is empty for argument %d", i)
		}
		if arg.DisplayTitle != "" {
			title = arg.DisplayTitle
		}
		switch arg.Type {
		case "string":
			rawAnswers[i] = ""
			fields = append(fields, huh.NewInput().Value(&rawAnswers[i]).Title(title).Description(arg.Description).Validate(
				func(s string) error {
					if arg.Regex != "" {
						matched, err := regexp.MatchString(arg.Regex, s)
						if err != nil {
							return fmt.Errorf("regex error: %w", err)
						}
						if !matched {
							return fmt.Errorf("input does not match the regex %s", arg.Regex)
						}
					}
					return nil
				}))
		case "int":
			rawAnswers[i] = ""
			fields = append(fields, huh.NewInput().Value(&rawAnswers[i]).Title(title).Description(arg.Description).Validate(
				func(s string) error {
					if arg.Regex != "" {
						matched, err := regexp.MatchString(arg.Regex, s)
						if err != nil {
							return fmt.Errorf("regex error: %w", err)
						}
						if !matched {
							return fmt.Errorf("input does not match the regex %s", arg.Regex)
						}
					}
					// Try to parse the int
					_, err := strconv.ParseInt(s, 10, 64)
					return err
				}))
		case "float":
			rawAnswers[i] = ""
			fields = append(fields, huh.NewInput().Value(&rawAnswers[i]).Title(title).Description(arg.Description).Validate(
				func(s string) error {
					if arg.Regex != "" {
						matched, err := regexp.MatchString(arg.Regex, s)
						if err != nil {
							return fmt.Errorf("regex error: %w", err)
						}
						if !matched {
							return fmt.Errorf("input does not match the regex %s", arg.Regex)
						}
					}

					// Try to parse the float
					_, err := strconv.ParseFloat(s, 64)
					return err
				}))
		case "bool":
			rawAnswers[i] = ""
			fields = append(fields, huh.NewInput().Value(&rawAnswers[i]).Title(title).Description(arg.Description).Validate(
				func(s string) error {
					_, err := strconv.ParseBool(s)
					if err != nil {
						// Check if the input is y, yes, n, no
						if s != "y" && s != "yes" && s != "n" && s != "no" {
							return fmt.Errorf("invalid boolean value. Expected y, yes, n, no, true, false, 1, 0, T, F, t, f")
						}
					}
					return nil
				}))

		default:
			return fmt.Errorf("unsupported argument type: %s", arg.Type)

		}
	}
	if len(fields) > 0 {
		group := huh.NewGroup(fields...)
		form := huh.NewForm(group)
		err = form.Run()
		if err != nil {
			return fmt.Errorf("failed to request the arguments: %w", err)
		}
	}

	// Convert the raw answers to the correct type
	answers := make([]interface{}, 0, len(manifest.Args))
	for i, arg := range manifest.Args {
		switch arg.Type {
		case "string":
			answers = append(answers, sql.Named(arg.Title, rawAnswers[i]))
		case "int":
			// Parse the int
			val, err := strconv.ParseInt(rawAnswers[i], 10, 64)
			if err != nil {
				return fmt.Errorf("failed to parse the int %s: %w", rawAnswers[i], err)
			}
			answers = append(answers, sql.Named(arg.Title, val))
		case "float":
			// Parse the float
			val, err := strconv.ParseFloat(rawAnswers[i], 64)
			if err != nil {
				return fmt.Errorf("failed to parse the float %s: %w", rawAnswers[i], err)
			}
			answers = append(answers, sql.Named(arg.Title, val))
		case "bool":
			// Parse the bool
			val, err := strconv.ParseBool(rawAnswers[i])
			if err != nil {
				rawAnswers[i] = strings.ToLower(rawAnswers[i])
				// Check if the input is y, yes, n, no
				if rawAnswers[i] == "y" || rawAnswers[i] == "yes" {
					val = true
				} else if rawAnswers[i] == "n" || rawAnswers[i] == "no" {
					val = false
				} else {
					return fmt.Errorf("failed to parse the bool %s: %w", rawAnswers[i], err)
				}
			}
			if val {
				answers = append(answers, sql.Named(arg.Title, 1))
			} else {
				answers = append(answers, sql.Named(arg.Title, 0))
			}
		default:
			return fmt.Errorf("unsupported argument type: %s", arg.Type)
		}
	}

	mapProfilePlugin := make(map[string]string)

	// For each plugin, if the plugin has multiple profiles, we have to ask the user which profile to use
	for _, plugin := range manifest.Plugins {
		// Sometimes, plugin can be downloaded from a different reg
		// We therefore have to check if the plugin is in the reg
		reg := "default"
		splitted := strings.Split(plugin, "/")
		if len(splitted) >= 2 {
			reg = splitted[0]
			plugin = splitted[1]
		}

		profiles, err := queries.GetProfilesOfPlugin(context.Background(), model.GetProfilesOfPluginParams{
			Pluginname: plugin,
			Registry:   reg,
		})
		if err != nil {
			return fmt.Errorf("could not get the profiles of the plugin: %w", err)
		}

		profileChosen := profiles[0].Name

		if len(profiles) > 1 {
			// Ask the user which profile to use
			profileNames := make([]string, 0, len(profiles))
			for _, profile := range profiles {
				profileNames = append(profileNames, profile.Name)
			}

			selectHuh := huh.NewSelect[string]().Title("Select the profile to use for the plugin " + plugin).Options(huh.NewOptions[string](profileNames...)...).
				Value(&profileChosen).Description("Because you have multiple profiles for the plugin " + plugin + ", you have to select the profile to use.")

			err = selectHuh.Run()
			if err != nil {
				return fmt.Errorf("failed to request the profile: %w", err)
			}
		}

		mapProfilePlugin[plugin] = profileChosen
	}

	// Get all the aliased tables
	aliases, err := queries.GetAliases(context.Background())
	if err != nil {
		return fmt.Errorf("could not get the plugin aliases: %w", err)
	}
	aliasesMap := make(map[string]string)
	for _, alias := range aliases {
		aliasesMap[alias.Tablename] = alias.Alias
	}

	// Create the shell
	shell := shell{
		DB: db,
		Middlewares: []middleware{
			middlewareSlashCommand, middlewareDotCommand,
			middlewarePRQL, middlewarePQL,
			middlewareMySQL, middlewareFileQuery,
			middlewareQuery,
		},
		Config: middlewareConfiguration{
			"dot-command":   true,
			"mysql":         true,
			"slash-command": true,
		},
		Namespace:      namespace,
		OutputFile:     "stdout",
		OutputFileDesc: os.Stdout,
	}

	// Check if stdout is a file
	outputFile, _ := cmd.Flags().GetString("output")
	if outputFile != "" {
		file, err := os.OpenFile(outputFile, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
		if err != nil {
			return fmt.Errorf("could not open output file: %w", err)
		}
		shell.OutputFile = outputFile
		shell.OutputFileDesc = file
	}

	// Check if the output file is a tty
	// If not, we set the output mode to plain
	if !term.IsTerminal(int(shell.OutputFileDesc.Fd())) {
		shell.Config["outputMode"] = "plain"
	}

	// Get the output mode if defined by the user
	outputFormat, _ := cmd.Flags().GetString("format")
	if outputFormat != "" {
		if _, ok := formatName[outputFormat]; ok {
			shell.Config["outputMode"] = outputFormat
		} else {
			return fmt.Errorf("invalid output format: %s", outputFormat)
		}
	}
	jsonOutput, _ := cmd.Flags().GetBool("json")
	if jsonOutput {
		shell.Config["outputMode"] = "json"
	}
	csvOutput, _ := cmd.Flags().GetBool("csv")
	if csvOutput {
		shell.Config["outputMode"] = "csv"
	}
	plainOutput, _ := cmd.Flags().GetBool("plain")
	if plainOutput {
		shell.Config["outputMode"] = "plain"
	}

	// Remove the manifest from the content
	content = regexpManifest.ReplaceAll(content, []byte(""))

	// Parse it and replace the table names
	// We have to replace the table names with the correct plugin and profile
	// Tables follows the following format: profile_plugin_table. Profiles named default are not included in the table name
	//
	// This issue exists let say the query template uses github_stargazers_from_repository but the user has two profile: default and work
	// While the template uses github_stargazers_from_repository, the user has to use work_github_stargazers_from_repository
	// Therefore, we rewrite the table names to include the profile (work in this case) if wished by the user
	queriesToRun := splitMultipleQuery(string(content))

	parser, err := sqlparser.New(sqlparser.Options{
		MySQLServerVersion: "8.0.23",
	})
	if err != nil {
		return fmt.Errorf("failed to create the SQL parser: %w", err)
	}

	for i, query := range queriesToRun {
		// Replace the table names
		stmt, err := parser.Parse(query)
		if err != nil {
			// A dot command or a comment
			continue
		}

		// Replace the table names
		sqlparser.Rewrite(stmt, nil, func(c *sqlparser.Cursor) bool {
			table, ok := c.Node().(sqlparser.TableName)
			if !ok {
				return true
			}

			tableName := table.Name.String()
			// Skip if the table name is empty
			if tableName == "" {
				return true
			}
			for plugin, profile := range mapProfilePlugin {
				if strings.HasPrefix(tableName, plugin) && profile != "default" {
					tableName = profile + "_" + tableName
					// Check if the table is aliased
					if alias, ok := aliasesMap[tableName]; ok {
						tableName = alias
					}
					break
				}
			}

			// Replace the table name
			c.Replace(sqlparser.TableName{
				Name:      sqlparser.NewIdentifierCS(tableName),
				Qualifier: table.Qualifier,
				Args:      table.Args,
			})

			return true
		})

		queriesToRun[i] = sqlparser.String(stmt)

	}

	// Run each query one by one
	for _, query := range queriesToRun {
		shell.Run(query, answers...)
	}

	return nil
}
