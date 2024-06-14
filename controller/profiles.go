package controller

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/charmbracelet/huh"
	"github.com/julien040/anyquery/controller/config/model"
	"github.com/julien040/anyquery/controller/config/registry"
	"github.com/spf13/cobra"
)

// Those validation functions are used to validate the input of the user
// They are very similar between each other, not very DRY, but it's okay
// Later, their business logic might change and they will diverge
func validateStringNotEmpty(s string) error {
	if s == "" {
		return fmt.Errorf("string is empty")
	}
	return nil
}

func validateStringIsNumber(s string) error {
	if _, err := strconv.Atoi(s); err != nil {
		return fmt.Errorf("string is not a number")
	}
	return nil
}

func validateStringIsFloat(s string) error {
	if _, err := strconv.ParseFloat(s, 64); err != nil {
		return fmt.Errorf("string is not a float")
	}
	return nil
}

func validateStringIsBool(s string) error {
	if _, err := strconv.ParseBool(s); err != nil {
		return fmt.Errorf("the value is not a boolean. Accepted values are: true, false, 1, 0, t, f, T, F")
	}
	return nil
}

func validateStringIsSliceOfInt(s string) error {
	parts := strings.Split(s, ",")
	for i, part := range parts {
		if _, err := strconv.Atoi(part); err != nil {
			return fmt.Errorf("the %dth part is not a number", i)
		}
	}
	return nil
}

func validateStringIsSliceOfFloat(s string) error {
	parts := strings.Split(s, ",")
	for i, part := range parts {
		if _, err := strconv.ParseFloat(part, 64); err != nil {
			return fmt.Errorf("the %dth part is not a float", i)
		}
	}
	return nil
}

func validateStringIsSliceOfBool(s string) error {
	parts := strings.Split(s, ",")
	for i, part := range parts {
		if _, err := strconv.ParseBool(part); err != nil {
			return fmt.Errorf("the %dth part is not a boolean", i)
		}
	}
	return nil
}

// Prompt the user to create a new profile or update an existing one if it exists
func createOrUpdateProfile(queries *model.Queries, registryName string, pluginName string, profileName string) error {
	// This function will ask the user for the configuration of a profile
	// If it already exists, it will update it
	// Otherwise, it will create it
	//
	// Due to the nature of the storage format, we need to convert between map and slice often in this function
	// Here is a summary of the variables to track, their usage and their type:
	// - toAskConfig: []registry.UserConfig => the required config by the plugin maker unserialized
	// - profileConfig: []interface{} => the slice of values of the profile configuration already parsed
	// and that will be passed to the form
	// - profileConfigTempString: []string => the slice of the raw values returned by the user in the form
	// and that need to be parsed to the correct type before storing them in profileConfig
	// - mapToUnserialize: map[string]interface{} => the map of the profile configuration already stored in the database
	// and that will be unserialized to profileConfig
	// - mapToSerialize: map[string]interface{} => the map of the profile configuration that will be serialized to the database
	// from the values in profileConfig
	//
	// To resume, the flow is:
	// - Get the required configuration from the plugin in toAskConfig
	// - Load the profile configuration from the database in mapToUnserialize
	// - Create a slice of values from mapToUnserialize in profileConfig (or from zero values if the profile doesn't exist)
	// - Create a slice of strings from profileConfig in profileConfigTempString so that we can fill the placeholders in the form
	// - Ask the user for the configuration (will fill profileConfigTempString)
	// - Convert the values in profileConfigTempString to the correct type and store them in profileConfig
	// - Transform profileConfig to a map in mapToSerialize
	// - Serialize mapToSerialize to JSON and store it in the database
	//
	// This function will return an error if the tty is not detected
	// or if the configuration is not valid
	//
	// This is quite a mess, feel free to refactor it if you have a better idea
	// Globally, this is due because the db uses a map while the form requires a pointer to a string
	// and we can't have a pointer to a map value

	if !isSTDinAtty() || !isSTDoutAtty() {
		return fmt.Errorf("no tty detected")
	}
	ctx := context.Background()
	// Get the plugin so that we can have the required configuration
	pluginInfo, err := queries.GetPlugin(ctx, model.GetPluginParams{
		Name:     pluginName,
		Registry: registryName,
	})
	if err != nil {
		return err
	}

	// Parse the asked configuration required by the plugin
	var toAskConfig []registry.UserConfig
	err = json.Unmarshal([]byte(pluginInfo.Config), &toAskConfig)
	if err != nil {
		return fmt.Errorf("could not parse the required configuration: %w", err)
	}

	// Store the profile configuration
	// and init it with zero values
	//
	// This will be later serialized to JSON and stored in the database
	profileConfig := make([]interface{}, len(toAskConfig))

	// Create a slice to store the temporary values of config as a string
	// This is used to validate the input before storing it in the profileConfig
	profileConfigTempString := make([]string, len(toAskConfig))

	for i, configKey := range toAskConfig {
		var value interface{}
		switch configKey.Type {
		case "string":
			value = ""
		case "int":
			value = 0
		case "float":
			value = 0.0
		case "bool":
			value = false
		case "[]string":
			value = make([]string, 0)
		case "[]int":
			value = make([]int, 0)
		case "[]float":
			value = make([]float64, 0)
		case "[]bool":
			value = make([]bool, 0)
		default:
			return fmt.Errorf("unknown type %s from plugin %s. Please ensure the registry is correct", configKey.Type, pluginName)
		}
		profileConfig[i] = value
		profileConfigTempString[i] = ""
	}

	profileExists := false

	// Now, if the profile exists, we load the configuration
	// so that we can fill the placeholders with the actual values
	// If it doesn't exist, we skip this step
	profile, err := queries.GetProfile(ctx, model.GetProfileParams{
		Name:       profileName,
		Pluginname: pluginName,
		Registry:   registryName,
	})
	if err == nil { // profile exists
		profileExists = true
		// Parse the profile configuration
		mapToUnserialize := make(map[string]interface{})
		err = json.Unmarshal([]byte(profile.Config), &mapToUnserialize)
		if err != nil {
			return fmt.Errorf("could not parse the profile configuration: %w", err)
		}

		// Now, we need to convert the map to a slice
		// so that we can fill the placeholders in the form
		for i, configKey := range toAskConfig {
			value, ok := mapToUnserialize[configKey.Name]
			// In case of an update, new fields might have been added
			// but aren't present in the previous configuration
			// so we skip them (they will later be filled by the user anyway)
			if ok {
				profileConfig[i] = value
			}
		}

		// Convert the parsed values to an unparsed string
		// that will be added as a placeholder in the form
		for i, value := range profileConfig {
			switch parsedVal := value.(type) {
			case string:
				profileConfigTempString[i] = parsedVal
			case int:
				profileConfigTempString[i] = strconv.Itoa(parsedVal)
			case float64:
				profileConfigTempString[i] = strconv.FormatFloat(parsedVal, 'f', -1, 64)
			case bool:
				profileConfigTempString[i] = strconv.FormatBool(parsedVal)
			case []interface{}:
				// encoding/json return []interface{} instead of the correct type
				build := strings.Builder{}
				for i, val := range parsedVal {
					switch val := val.(type) {
					case string:
						build.WriteString(val)
					case int:
						build.WriteString(strconv.Itoa(val))
					case float64:
						build.WriteString(strconv.FormatFloat(val, 'f', -1, 64))
					case bool:
						build.WriteString(strconv.FormatBool(val))
					default:
						return fmt.Errorf("unknown type %T of value %v from profile %s. Please ensure the data in the database wasn't modified by external software", val, val, profileName)
					}
					if i != len(parsedVal)-1 {
						build.WriteString(",")
					}
				}
				profileConfigTempString[i] = build.String()
			default:
				return fmt.Errorf("unknown type %T of value %v from profile %s. Please ensure the data in the database wasn't modified by external software", value, value, profileName)
			}
		}
	}

	// Ask the user for the configuration
	// To do so, we will build a form using the charmbracelet/huh package
	groupForm := make([]*huh.Group, len(toAskConfig))
	for i, configKey := range toAskConfig {

		title := configKey.Name
		if configKey.Required {
			title += "*"
		}
		title += " (type: " + configKey.Type + ")"
		input := huh.NewInput().Title(title).
			Value(&(profileConfigTempString[i]))

		var description string
		if configKey.Required {
			description = "This field is required"
		} else {
			description = "This field is optional"
		}

		if configKey.Type == "[]string" || configKey.Type == "[]int" || configKey.Type == "[]float" || configKey.Type == "[]bool" {
			description += " and takes a comma-separated list of values (e.g. value1,value2,value3)"
		}

		input.Description(description)

		// Set the validation function based on the type
		switch configKey.Type {
		case "string", "[]string":
			if configKey.Required {
				input.Validate(validateStringNotEmpty)
			}
		case "int":
			// If required, will fail on empty string
			if configKey.Required {
				input.Validate(validateStringNotEmpty)
			} else {
				input.Validate(func(s string) error {
					if s == "" {
						return nil
					}
					return validateStringIsNumber(s)
				})
			}
		case "float":
			// If required, will fail on empty string
			if configKey.Required {
				input.Validate(validateStringNotEmpty)
			} else {
				input.Validate(func(s string) error {
					if s == "" {
						return nil
					}
					return validateStringIsFloat(s)
				})
			}
		case "bool":
			// If required, will fail on empty string
			if configKey.Required {
				input.Validate(validateStringNotEmpty)
			} else {
				input.Validate(func(s string) error {
					if s == "" {
						return nil
					} else {
						return validateStringIsBool(s)
					}
				})
			}
		case "[]int":
			// If required, will fail on empty string
			if configKey.Required {
				input.Validate(func(s string) error {
					if s == "" {
						return fmt.Errorf("the field is required")
					}
					return validateStringIsSliceOfInt(s)
				})
			} else {
				input.Validate(validateStringIsSliceOfInt)
			}
		case "[]float":
			// If required, will fail on empty string
			if configKey.Required {
				input.Validate(func(s string) error {
					if s == "" {
						return fmt.Errorf("the field is required")
					}
					return validateStringIsSliceOfFloat(s)
				})
			} else {
				input.Validate(validateStringIsSliceOfFloat)
			}
		case "[]bool":
			// If required, will fail on empty string
			if configKey.Required {
				input.Validate(func(s string) error {
					if s == "" {
						return fmt.Errorf("the field is required")
					}
					return validateStringIsSliceOfBool(s)
				})
			} else {
				input.Validate(validateStringIsSliceOfBool)
			}
		default:
			return fmt.Errorf("unknown type %s from plugin %s while preparing the form. Please ensure the registry is correct", configKey.Type, pluginName)
		}

		groupForm[i] = huh.NewGroup(input)
	}

	if profileExists {
		fmt.Println("💪 Let's update the profile", profileName, "for the plugin", pluginName)
	} else if profileName == "" {
		fmt.Println("💪 Let's configure the plugin", pluginName)
	} else {
		fmt.Println("💪 Let's create a new profile", profileName, "for the plugin", pluginName)
	}

	// Create the form
	form := huh.NewForm(groupForm...)
	err = form.Run()
	if err != nil {
		// If the user pressed CTRL+C, we don't do anything
		if err.Error() == "user aborted" {
			return nil
		}
		return fmt.Errorf("could not run the form: %w", err)
	}

	// Now, we have the values in profileConfigTempString
	// so we need to convert them to the correct type
	// and then store them serialized in the database
	for i, configVal := range profileConfig {
		switch configVal.(type) {
		case string:
			profileConfig[i] = profileConfigTempString[i]
		case int:
			if profileConfigTempString[i] == "" {
				profileConfig[i] = 0
			} else {
				profileConfig[i], err = strconv.Atoi(profileConfigTempString[i])
				if err != nil {
					return fmt.Errorf("could not convert the value to int: %w", err)
				}
			}

		case float64:
			if profileConfigTempString[i] == "" {
				profileConfig[i] = 0.0
			} else {
				profileConfig[i], err = strconv.ParseFloat(profileConfigTempString[i], 64)
				if err != nil {
					return fmt.Errorf("could not convert the value to float: %w", err)
				}
			}
		case bool:
			if profileConfigTempString[i] == "" {
				profileConfig[i] = false
			} else {
				profileConfig[i], err = strconv.ParseBool(profileConfigTempString[i])
				if err != nil {
					return fmt.Errorf("could not convert the value to bool: %w", err)
				}
			}
		// encoding/json return []interface{} instead of the correct type
		// It means we have to repeat the same code as below
		case []interface{}:
			parts := strings.Split(profileConfigTempString[i], ",")
			switch toAskConfig[i].Type {
			case "[]string":
				profileConfig[i] = parts
			case "[]int":
				ints := make([]int, len(parts))
				for i, part := range parts {
					ints[i], err = strconv.Atoi(part)
					if err != nil {
						return fmt.Errorf("could not convert the value %s to int: %w", part, err)
					}
				}
				profileConfig[i] = ints
			case "[]float":
				floats := make([]float64, len(parts))
				for i, part := range parts {
					floats[i], err = strconv.ParseFloat(part, 64)
					if err != nil {
						return fmt.Errorf("could not convert the value %s to float: %w", part, err)
					}
				}
				profileConfig[i] = floats

			case "[]bool":
				bools := make([]bool, len(parts))
				for i, part := range parts {
					bools[i], err = strconv.ParseBool(part)
					if err != nil {
						return fmt.Errorf("could not convert the value %s to bool: %w", part, err)
					}
				}

				profileConfig[i] = bools
			}

		case []string:
			if profileConfigTempString[i] == "" {
				profileConfig[i] = []string{}
			} else {
				profileConfig[i] = strings.Split(profileConfigTempString[i], ",")
			}

		case []int:
			if profileConfigTempString[i] == "" {
				profileConfig[i] = []int{}
			} else {
				parts := strings.Split(profileConfigTempString[i], ",")
				ints := make([]int, len(parts))
				for i, part := range parts {
					ints[i], err = strconv.Atoi(part)
					if err != nil {
						return fmt.Errorf("could not convert the value to int: %w", err)
					}
				}
				profileConfig[i] = ints
			}
		case []float64:
			if profileConfigTempString[i] == "" {
				profileConfig[i] = []float64{}
			} else {
				parts := strings.Split(profileConfigTempString[i], ",")
				floats := make([]float64, len(parts))
				for i, part := range parts {
					floats[i], err = strconv.ParseFloat(part, 64)
					if err != nil {
						return fmt.Errorf("could not convert the value to float: %w", err)
					}
				}
				profileConfig[i] = floats
			}
		case []bool:
			if profileConfigTempString[i] == "" {
				profileConfig[i] = []bool{}
			} else {
				parts := strings.Split(profileConfigTempString[i], ",")
				bools := make([]bool, len(parts))
				for i, part := range parts {
					bools[i], err = strconv.ParseBool(part)
					if err != nil {
						return fmt.Errorf("could not convert the value to bool: %w", err)
					}
				}
				profileConfig[i] = bools
			}

		default:
			return fmt.Errorf("unknown type %T of value %t from profile %s at index %d while parsing values. Please ensure the registry is correct", configVal, configVal, profileName, i)
		}

	}

	// Serialize the profile configuration
	mapToSerialize := make(map[string]interface{})
	for i, configKey := range toAskConfig {
		mapToSerialize[configKey.Name] = profileConfig[i]
	}
	profileConfigJSON, err := json.Marshal(mapToSerialize)
	if err != nil {
		return fmt.Errorf("could not serialize the profile configuration: %w", err)
	}

	// Create or update the profile
	if profileExists {
		err = queries.UpdateProfileConfig(ctx, model.UpdateProfileConfigParams{
			Config:     string(profileConfigJSON),
			Name:       profileName,
			Pluginname: pluginName,
			Registry:   registryName,
		})
		if err != nil {
			return fmt.Errorf("could not update the profile to the database: %w", err)
		}
	} else {
		err = queries.AddProfile(ctx, model.AddProfileParams{
			Config:     string(profileConfigJSON),
			Name:       profileName,
			Pluginname: pluginName,
			Registry:   registryName,
		})
		if err != nil {
			return fmt.Errorf("could not add the profile to the database: %w", err)
		}
	}

	if profileExists {
		fmt.Println("✅ Successfully updated profile", profileName)
	} else {
		fmt.Println("✅ Successfully created profile", profileName)
	}

	return nil

}

func ProfileUpdate(cmd *cobra.Command, args []string) error {
	// Open the database on read-write mode
	db, querier, err := requestDatabase(cmd.Flags(), false)
	if err != nil {
		return fmt.Errorf("could not open the database: %w", err)
	}
	defer db.Close()

	var registry = "default"
	var pluginName = ""
	var profileName = ""

	if len(args) == 3 {
		registry = args[0]
		pluginName = args[1]
		profileName = args[2]
	} else if len(args) == 2 {
		pluginName = args[0]
		profileName = args[1]
	} else {
		// We will prompt the user to select the registry
		// and then the plugin
		// and then the profile name
		// This is a bit more user-friendly than the CLI

		// Select the registry
		registryInput, err := selectRegistry(querier, &registry)
		if err != nil {
			return fmt.Errorf("could not select the registry: %w", err)
		}
		err = registryInput.Run()
		if err != nil {
			if err.Error() == "user aborted" {
				return nil
			}
			return fmt.Errorf("could not run the form: %w", err)
		}

		// Select the plugin
		pluginInput, err := selectPlugin(querier, registry, &pluginName)
		if err != nil {
			return fmt.Errorf("could not select the plugin: %w", err)
		}

		err = pluginInput.Run()
		if err != nil {
			if err.Error() == "user aborted" {
				return nil
			}
			return fmt.Errorf("could not run the form: %w", err)
		}

		profileInput, err := selectProfile(querier, registry, pluginName, &profileName)
		if err != nil {
			return fmt.Errorf("could not select the profile: %w", err)
		}

		// Ask for the profile name
		err = profileInput.Run()
		if err != nil {
			// CTRL+C so we don't do anything
			if err.Error() == "user aborted" {
				return nil
			}
			return fmt.Errorf("could not run the form: %w", err)
		}

	}

	// Check if the plugin exists
	ctx := context.Background()
	_, err = querier.GetPlugin(ctx, model.GetPluginParams{
		Name:     pluginName,
		Registry: registry,
	})
	if err != nil {
		if err.Error() == "sql: no rows in result set" {
			return fmt.Errorf("a plugin named %s in the registry %s is not installed", pluginName, registry)
		}
		return fmt.Errorf("could not get the registry: %w", err)
	}

	// Check if the profile exists
	_, err = querier.GetProfile(ctx, model.GetProfileParams{
		Registry:   registry,
		Pluginname: pluginName,
		Name:       profileName,
	})

	if err != nil {
		if err.Error() == "sql: no rows in result set" {
			return fmt.Errorf("a profile with the name %s for the plugin %s does not exist", profileName, pluginName)
		}
		return fmt.Errorf("could not get the profile: %w", err)
	}

	// We refresh the profile
	// createOrUpdateProfile will ask the user for the configuration
	// and print any error that might occur
	// There is no need to check the error here
	return createOrUpdateProfile(querier, registry, pluginName, profileName)
}

func ProfileNew(cmd *cobra.Command, args []string) error {
	// Open the database on read-write mode
	db, querier, err := requestDatabase(cmd.Flags(), false)
	if err != nil {
		return fmt.Errorf("could not open the database: %w", err)
	}
	defer db.Close()

	var registry = "default"
	var pluginName = ""
	var profileName = ""

	if len(args) == 3 {
		registry = args[0]
		pluginName = args[1]
		profileName = args[2]
	} else if len(args) == 2 {
		// If there are only two arguments, we assume the default registry
		pluginName = args[0]
		profileName = args[1]
	} else {
		// We will prompt the user to select the registry, a plugin and then a profile
		// Because form values are based on precedent values, we can't use a form
		// Therefore, we will ask each question one by one

		// Select the registry
		registryInput, err := selectRegistry(querier, &registry)
		if err != nil {
			return fmt.Errorf("could not select the registry: %w", err)
		}
		err = registryInput.Run()
		if err != nil {
			if err.Error() == "user aborted" {
				return nil
			}
			return fmt.Errorf("could not run the form: %w", err)
		}

		// Select the plugin
		pluginInput, err := selectPlugin(querier, registry, &pluginName)
		if err != nil {
			return fmt.Errorf("could not select the plugin: %w", err)
		}

		err = pluginInput.Run()
		if err != nil {
			if err.Error() == "user aborted" {
				return nil
			}
			return fmt.Errorf("could not run the form: %w", err)
		}

		profileInput := huh.NewInput().Title("Profile name").Value(&profileName).
			Description("The table will be prefixed with the plugin name. For example, if the plugin is 'myplugin' and the profile is 'profile1', the table will be 'profile1_myplugin_table'")
		err = profileInput.Run()
		if err != nil {
			// CTRL+C so we don't do anything
			if err.Error() == "user aborted" {
				return nil
			}
			return fmt.Errorf("could not run the form: %w", err)
		}
	}

	// Check if the plugin exists
	ctx := context.Background()
	_, err = querier.GetPlugin(ctx, model.GetPluginParams{
		Name:     pluginName,
		Registry: registry,
	})
	if err != nil {
		if err.Error() == "sql: no rows in result set" {
			return fmt.Errorf("a plugin named %s in the registry %s is not installed", pluginName, registry)
		}
		return fmt.Errorf("could not get the registry: %w", err)
	}

	// Check if the profile exists
	_, err = querier.GetProfile(ctx, model.GetProfileParams{
		Registry:   registry,
		Pluginname: pluginName,
		Name:       profileName,
	})

	if err == nil {
		return fmt.Errorf("a profile with the name %s for the plugin %s already exists", profileName, pluginName)
	}

	// Check if the plugin exists
	_, err = querier.GetPlugin(ctx, model.GetPluginParams{
		Name:     pluginName,
		Registry: registry,
	})
	if err != nil {
		if err.Error() == "sql: no rows in result set" {
			return fmt.Errorf("a plugin named %s in the registry %s is not installed", pluginName, registry)
		}
		return fmt.Errorf("could not get the plugin: %w", err)
	}

	// We create a new profile
	return createOrUpdateProfile(querier, registry, pluginName, profileName)
}

func ProfileList(cmd *cobra.Command, args []string) error {
	// Open the database on read-only mode
	db, querier, err := requestDatabase(cmd.Flags(), true)
	if err != nil {
		return fmt.Errorf("could not open the database: %w", err)
	}
	defer db.Close()

	var registry = ""
	var plugin = ""
	if len(args) == 2 {
		registry = args[0]
		plugin = args[1]
	} else if len(args) == 1 {
		registry = args[0]
	}

	var profiles []model.Profile

	ctx := context.Background()
	if registry != "" && plugin != "" {
		profiles, err = querier.GetProfilesOfPlugin(ctx, model.GetProfilesOfPluginParams{
			Pluginname: plugin,
			Registry:   registry,
		})
	} else if registry != "" && plugin == "" {
		profiles, err = querier.GetProfilesOfRegistry(ctx, registry)
	} else {
		profiles, err = querier.GetProfiles(ctx)
	}

	if err != nil {
		return fmt.Errorf("could not get the profiles: %w", err)
	}
	output := outputTable{
		Writer:  os.Stdout,
		Columns: []string{"Profile name", "Plugin", "Registry"},
	}

	output.InferFlags(cmd.Flags())
	for _, profile := range profiles {
		err = output.Write([]interface{}{profile.Name, profile.Pluginname, profile.Registry})
		if err != nil {
			return fmt.Errorf("could not write the profile %s to the output: %w", profile.Name, err)
		}
	}
	return output.Close()

}

func ProfileDelete(cmd *cobra.Command, args []string) error {
	// Open the database on read-write mode
	db, querier, err := requestDatabase(cmd.Flags(), false)
	if err != nil {
		return fmt.Errorf("could not open the database: %w", err)
	}
	defer db.Close()

	var registry = "default"
	var plugin = ""
	var profile = ""

	if len(args) == 3 {
		registry = args[0]
		plugin = args[1]
		profile = args[2]
	} else if len(args) == 2 {
		plugin = args[0]
		profile = args[1]
	} else {
		// We will prompt the user to select the registry, a plugin and then a profile
		// Because form values are based on precedent values, we can't use a form
		// Therefore, we will ask each question one by one

		// Select the registry
		registryInput, err := selectRegistry(querier, &registry)
		if err != nil {
			return fmt.Errorf("could not select the registry: %w", err)
		}
		err = registryInput.Run()
		if err != nil {
			if err.Error() == "user aborted" {
				return nil
			}
			return fmt.Errorf("could not run the form: %w", err)
		}

		// Select the plugin
		pluginInput, err := selectPlugin(querier, registry, &plugin)
		if err != nil {
			return fmt.Errorf("could not select the plugin: %w", err)
		}

		err = pluginInput.Run()
		if err != nil {
			if err.Error() == "user aborted" {
				return nil
			}
			return fmt.Errorf("could not run the form: %w", err)
		}

		profileInput, err := selectProfile(querier, registry, plugin, &profile)
		if err != nil {
			return fmt.Errorf("could not select the profile: %w", err)
		}

		// Ask for the profile name
		err = profileInput.Run()
		if err != nil {
			// CTRL+C so we don't do anything
			if err.Error() == "user aborted" {
				return nil
			}
			return fmt.Errorf("could not run the form: %w", err)
		}
	}

	// Check if the profile exists
	// We don't need to check if the plugin exists nor the registry

	ctx := context.Background()
	_, err = querier.GetProfile(ctx, model.GetProfileParams{
		Registry:   registry,
		Pluginname: plugin,
		Name:       profile,
	})

	if err != nil {
		if err.Error() == "sql: no rows in result set" {
			return fmt.Errorf("a profile with the name %s for the plugin %s does not exist", profile, plugin)
		}
		return fmt.Errorf("could not get the profile: %w", err)
	}

	err = querier.DeleteProfile(ctx, model.DeleteProfileParams{
		Name:       profile,
		Pluginname: plugin,
		Registry:   registry,
	})
	if err != nil {
		return fmt.Errorf("could not delete the profile: %w", err)
	}

	fmt.Println("✅ Successfully deleted the profile", profile, "for the plugin", plugin)

	return nil
}
