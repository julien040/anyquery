package main

import (
	"github.com/brianvoe/gofakeit/v7"
	"github.com/julien040/anyquery/rpc"
)

// A constructor to create a new table instance
// This function is called everytime a new connection is made to the plugin
//
// It should return a new table instance, the database schema and if there is an error
func peopleCreator(args rpc.TableCreatorArgs) (rpc.Table, *rpc.DatabaseSchema, error) {
	return &peopleTable{}, &rpc.DatabaseSchema{
		PrimaryKey: -1,
		Columns: []rpc.DatabaseSchemaColumn{
			{
				Name:        "id",
				Type:        rpc.ColumnTypeInt,
				Description: "The ID of the row",
			},
			{
				Name:        "first_name",
				Type:        rpc.ColumnTypeString,
				Description: "A random first name",
			},
			{
				Name:        "last_name",
				Type:        rpc.ColumnTypeString,
				Description: "A random last name",
			},
			{
				Name:        "gender",
				Type:        rpc.ColumnTypeString,
				Description: "A random gender",
			},
			{
				Name:        "ssn",
				Type:        rpc.ColumnTypeString,
				Description: "A random social security number",
			},
			{
				Name:        "hobby",
				Type:        rpc.ColumnTypeString,
				Description: "A random hobby",
			},
			{
				Name:        "job_company",
				Type:        rpc.ColumnTypeString,
				Description: "A random job company",
			},
			{
				Name:        "job_title",
				Type:        rpc.ColumnTypeString,
				Description: "A random job title",
			},
			{
				Name:        "address",
				Type:        rpc.ColumnTypeString,
				Description: "A random address",
			},
			{
				Name:        "street",
				Type:        rpc.ColumnTypeString,
				Description: "A random street",
			},
			{
				Name:        "city",
				Type:        rpc.ColumnTypeString,
				Description: "A random city",
			},
			{
				Name:        "state",
				Type:        rpc.ColumnTypeString,
				Description: "A random state",
			},
			{
				Name:        "zip",
				Type:        rpc.ColumnTypeString,
				Description: "A random zip code",
			},
			{
				Name:        "country",
				Type:        rpc.ColumnTypeString,
				Description: "A random country",
			},
			{
				Name:        "latitude",
				Type:        rpc.ColumnTypeFloat,
				Description: "A random latitude",
			},
			{
				Name:        "longitude",
				Type:        rpc.ColumnTypeFloat,
				Description: "A random longitude",
			},
			{
				Name:        "phone",
				Type:        rpc.ColumnTypeString,
				Description: "A random phone number",
			},
			{
				Name:        "email",
				Type:        rpc.ColumnTypeString,
				Description: "A random email",
			},
			{
				Name:        "credit_card_number",
				Type:        rpc.ColumnTypeString,
				Description: "A random credit card number",
			},
			{
				Name:        "credit_card_type",
				Type:        rpc.ColumnTypeString,
				Description: "A random credit card type",
			},
			{
				Name:        "credit_card_expiration",
				Type:        rpc.ColumnTypeString,
				Description: "A random credit card expiration",
			},
			{
				Name:        "credit_card_cvv",
				Type:        rpc.ColumnTypeInt,
				Description: "A random credit card CVV",
			},
			{
				Name:        "username",
				Type:        rpc.ColumnTypeString,
				Description: "A random username",
			},
			{
				Name:        "password",
				Type:        rpc.ColumnTypeString,
				Description: "A random password",
			},
			{
				Name:        "favorite_beer",
				Type:        rpc.ColumnTypeString,
				Description: "A random favorite beer",
			},
			{
				Name:        "car_maker",
				Type:        rpc.ColumnTypeString,
				Description: "A random car maker",
			},
			{
				Name:        "car_model",
				Type:        rpc.ColumnTypeString,
				Description: "A random car model",
			},
			{
				Name:        "car_type",
				Type:        rpc.ColumnTypeString,
				Description: "A random car type",
			},
			{
				Name:        "car_transmission",
				Type:        rpc.ColumnTypeString,
				Description: "A random car transmission",
			},
			{
				Name:        "car_fuel",
				Type:        rpc.ColumnTypeString,
				Description: "A random car fuel",
			},
			{
				Name:        "favorite_fruit",
				Type:        rpc.ColumnTypeString,
				Description: "A random favorite fruit",
			},
			{
				Name:        "favorite_vegetable",
				Type:        rpc.ColumnTypeString,
				Description: "A random favorite vegetable",
			},
			{
				Name:        "uuid",
				Type:        rpc.ColumnTypeString,
				Description: "A random UUID",
			},
			{
				Name:        "favorite_color",
				Type:        rpc.ColumnTypeString,
				Description: "A random favorite color",
			},
			{
				Name:        "favorite_color_hex",
				Type:        rpc.ColumnTypeString,
				Description: "A random favorite color hex",
			},
			{
				Name:        "pet_name",
				Type:        rpc.ColumnTypeString,
				Description: "A random pet name",
			},
			{
				Name:        "pet_type",
				Type:        rpc.ColumnTypeString,
				Description: "A random pet type",
			},
			{
				Name:        "language_spoken",
				Type:        rpc.ColumnTypeString,
				Description: "A random language spoken",
			},
			{
				Name:        "ƒavorite_programming_language",
				Type:        rpc.ColumnTypeString,
				Description: "A random favorite programming language",
			},
			{
				Name:        "favorite_sport_player",
				Type:        rpc.ColumnTypeString,
				Description: "A random favorite sport player",
			},
			{
				Name:        "favorite_actor",
				Type:        rpc.ColumnTypeString,
				Description: "A random favorite actor",
			},
			{
				Name:        "favorite_movie",
				Type:        rpc.ColumnTypeString,
				Description: "A random favorite movie",
			},
			{
				Name:        "favorite_book",
				Type:        rpc.ColumnTypeString,
				Description: "A random favorite book",
			},
		},
	}, nil
}

type peopleTable struct {
}

type peopleCursor struct {
	rowID int64
}

// Return a slice of rows that will be returned to Anyquery and filtered.
// The second return value is true if the cursor has no more rows to return
//
// The constraints are used for optimization purposes to "pre-filter" the rows
// If the rows returned don't match the constraints, it's not an issue. Anyquery will filter them out
func (t *peopleCursor) Query(constraints rpc.QueryConstraint) ([][]interface{}, bool, error) {
	// Return 1000 rows per call
	rows := make([][]interface{}, 0, 1000)

	// Generate 1000 rows
	for i := 0; i < 1000; i++ {
		person := gofakeit.Person()
		car := gofakeit.Car()
		rows = append(rows, []interface{}{
			t.rowID + int64(i),
			person.FirstName,
			person.LastName,
			person.Gender,
			person.SSN,
			person.Hobby,
			person.Job.Company,
			person.Job.Title,
			person.Address.Address,
			person.Address.Street,
			person.Address.City,
			person.Address.State,
			person.Address.Zip,
			person.Address.Country,
			person.Address.Latitude,
			person.Address.Longitude,
			person.Contact.Phone,
			person.Contact.Email,
			person.CreditCard.Number,
			person.CreditCard.Type,
			person.CreditCard.Exp,
			person.CreditCard.Cvv,
			person.FirstName + gofakeit.LetterN(4),
			gofakeit.Password(true, true, true, false, false, 12),
			gofakeit.BeerName(),
			car.Brand,
			car.Model,
			car.Type,
			car.Transmission,
			car.Fuel,
			gofakeit.Fruit(),
			gofakeit.Vegetable(),
			gofakeit.UUID(),
			gofakeit.Color(),
			gofakeit.HexColor(),
			gofakeit.PetName(),
			gofakeit.Animal(),
			gofakeit.Language(),
			gofakeit.ProgrammingLanguage(),
			gofakeit.CelebritySport(),
			gofakeit.CelebrityActor(),
			gofakeit.Movie().Name,
			gofakeit.Book().Title,
		})

	}

	t.rowID += 1000

	// Return the rows
	return rows, false, nil
}

// Create a new cursor that will be used to read rows
func (t *peopleTable) CreateReader() rpc.ReaderInterface {
	return &peopleCursor{
		rowID: 0,
	}
}

// A slice of rows to insert
func (t *peopleTable) Insert(rows [][]interface{}) error {
	return nil
}

// A slice of rows to update
// The first element of each row is the primary key
// while the rest are the values to update
// The primary key is therefore present twice
func (t *peopleTable) Update(rows [][]interface{}) error {
	return nil
}

// A slice of primary keys to delete
func (t *peopleTable) Delete(primaryKeys []interface{}) error {
	return nil
}

// A destructor to clean up resources
func (t *peopleTable) Close() error {
	return nil
}
