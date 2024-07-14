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
				Name: "id",
				Type: rpc.ColumnTypeInt,
			},
			{
				Name: "first_name",
				Type: rpc.ColumnTypeString,
			},
			{
				Name: "last_name",
				Type: rpc.ColumnTypeString,
			},
			{
				Name: "gender",
				Type: rpc.ColumnTypeString,
			},
			{
				Name: "ssn",
				Type: rpc.ColumnTypeString,
			},
			{
				Name: "hobby",
				Type: rpc.ColumnTypeString,
			},
			{
				Name: "job_company",
				Type: rpc.ColumnTypeString,
			},
			{
				Name: "job_title",
				Type: rpc.ColumnTypeString,
			},
			{
				Name: "address",
				Type: rpc.ColumnTypeString,
			},
			{
				Name: "street",
				Type: rpc.ColumnTypeString,
			},
			{
				Name: "city",
				Type: rpc.ColumnTypeString,
			},
			{
				Name: "state",
				Type: rpc.ColumnTypeString,
			},
			{
				Name: "zip",
				Type: rpc.ColumnTypeString,
			},
			{
				Name: "country",
				Type: rpc.ColumnTypeString,
			},
			{
				Name: "latitude",
				Type: rpc.ColumnTypeFloat,
			},
			{
				Name: "longitude",
				Type: rpc.ColumnTypeFloat,
			},
			{
				Name: "phone",
				Type: rpc.ColumnTypeString,
			},
			{
				Name: "email",
				Type: rpc.ColumnTypeString,
			},
			{
				Name: "credit_card_number",
				Type: rpc.ColumnTypeString,
			},
			{
				Name: "credit_card_type",
				Type: rpc.ColumnTypeString,
			},
			{
				Name: "credit_card_expiration",
				Type: rpc.ColumnTypeString,
			},
			{
				Name: "credit_card_cvv",
				Type: rpc.ColumnTypeInt,
			},
			{
				Name: "username",
				Type: rpc.ColumnTypeString,
			},
			{
				Name: "password",
				Type: rpc.ColumnTypeString,
			},
			{
				Name: "favorite_beer",
				Type: rpc.ColumnTypeString,
			},
			{
				Name: "car_maker",
				Type: rpc.ColumnTypeString,
			},
			{
				Name: "car_model",
				Type: rpc.ColumnTypeString,
			},
			{
				Name: "car_type",
				Type: rpc.ColumnTypeString,
			},
			{
				Name: "car_transmission",
				Type: rpc.ColumnTypeString,
			},
			{
				Name: "car_fuel",
				Type: rpc.ColumnTypeString,
			},
			{
				Name: "favorite_fruit",
				Type: rpc.ColumnTypeString,
			},
			{
				Name: "favorite_vegetable",
				Type: rpc.ColumnTypeString,
			},
			{
				Name: "uuid",
				Type: rpc.ColumnTypeString,
			},
			{
				Name: "favorite_color",
				Type: rpc.ColumnTypeString,
			},
			{
				Name: "favorite_color_hex",
				Type: rpc.ColumnTypeString,
			},
			{
				Name: "pet_name",
				Type: rpc.ColumnTypeString,
			},
			{
				Name: "pet_type",
				Type: rpc.ColumnTypeString,
			},
			{
				Name: "language_spoken",
				Type: rpc.ColumnTypeString,
			},
			{
				Name: "ƒavorite_programming_language",
				Type: rpc.ColumnTypeString,
			},
			{
				Name: "favorite_sport_player",
				Type: rpc.ColumnTypeString,
			},
			{
				Name: "favorite_actor",
				Type: rpc.ColumnTypeString,
			},
			{
				Name: "favorite_movie",
				Type: rpc.ColumnTypeString,
			},
			{
				Name: "favorite_book",
				Type: rpc.ColumnTypeString,
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
