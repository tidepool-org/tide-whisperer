package store

import (
	"context"
	"errors"
	"fmt"
	"os"
	"reflect"
	"time"

	"github.com/go-pg/pg/v10"
	"github.com/go-pg/pg/v10/orm"
	"github.com/tidepool-org/tide-whisperer/models"
)

type (
	// TimeseriesStoreClient - Timeseries Storage Client
	TimeseriesStoreClient struct {
		db   *pg.DB
		context  context.Context
		//database string
	}

	tsIterator struct {
		results []interface{}
		pos     int
	}
)

var (
	DBContextTimeout = time.Duration(20)*time.Second
)

func init() {
	orm.SetTableNameInflector(func(s string) string {
		return  s
	})
}

func NewDbContext() context.Context {
	ctx, _ := context.WithTimeout(context.Background(), DBContextTimeout)
	return ctx
}

type dbLogger struct { }

func (d dbLogger) BeforeQuery(c context.Context, q *pg.QueryEvent) (context.Context, error) {
	return c, nil
}

func (d dbLogger) AfterQuery(c context.Context, q *pg.QueryEvent) error {
	b, _ := q.FormattedQuery()
	s := string(b)
	fmt.Println("Query: ", s)
	return nil
}

// NewTimeseriesStoreClient creates a new TimeseriesStoreClient
func NewTimeseriesStoreClient() *TimeseriesStoreClient {

	// Connect to db
	user, _ := os.LookupEnv("TIMESCALEDB_USER")
	password, _ := os.LookupEnv("TIMESCALEDB_PASSWORD")
	host, _ := os.LookupEnv("TIMESCALEDB_HOST")
	db_name, _ := os.LookupEnv("TIMESCALEDB_DBNAME")


	url := fmt.Sprintf("postgres://%s:%s@%s:5432/%s?sslmode=allow", user, password, host, db_name)
	//url := fmt.Sprintf("postgres://%s:%s@%s:5432/%s?sslmode=disable", user, password, host, db_name)
	opt, err := pg.ParseURL(url)
	if err != nil {
		panic(err)
	}

	db := pg.Connect(opt)
	fmt.Println("Trying to connect to db")

	ctx := NewDbContext()

	db.AddQueryHook(dbLogger{})


	// Check if connection credentials are valid and PostgreSQL is up and running.
	if err := db.Ping(ctx); err != nil {
		fmt.Println("Error: ", err)
		return nil
	}
	fmt.Println("Connected successfully")

	return &TimeseriesStoreClient{
		db:   db,
		context:  context.Background(),
	}
}

// WithContext returns a shallow copy of c with its context changed
// to ctx. The provided ctx must be non-nil.
func (c *TimeseriesStoreClient) WithContext(ctx context.Context) *TimeseriesStoreClient{
	if ctx == nil {
		panic("nil context")
	}
	c2 := new(TimeseriesStoreClient)
	*c2 = *c
	c2.context = ctx
	return c2
}

// EnsureIndexes exist for the TimeseriesDB collection. EnsureIndexes uses the Background() context, in order
// to pass back the TimeseriesDB errors, rather than any context errors.
func (t *TimeseriesStoreClient) EnsureIndexes() error {
	return nil
}

// Ping the TimeseriesDB database
func (t *TimeseriesStoreClient) Ping() error {
	// do we have a store session
	t.db.Ping(t.context)
	return nil
}

// Disconnect from the TimeseriesDB database
func (t *TimeseriesStoreClient) Disconnect() error {
	return t.db.Close()
}

// HasMedtronicDirectData - check whether the userID has Medtronic data that has been uploaded via Uploader
/*
func (t *TimeseriesStoreClient) HasMedtronicDirectData(userID string) (bool, error) {
	model := new ([]models.Upload)
	query := t.db.Model(model)
	query = query.
		Where("user_id = ?", userID).
		Where("state = closed").
		//Where("active = ?", true).
		//Where("deletedTime != ?", nil).
		Where("deviceManufacturers = Medtronic")


	resultErr := query.Select()
	if resultErr != nil {
		return false, resultErr
	}

	return len(*model) > 0, nil
}

// GetDexcomDataSource - get
func (t *TimeseriesStoreClient) GetDexcomDataSource(userID string) (bson.M, error) {
	if userID == "" {
		return nil, errors.New("user id is missing")
	}

	// `earliestDataTime` and `latestDataTime` are bson.Date fields. Internally, they are int64's
	// so if they exist, the must be set to something, even if 0 (ie Unix epoch)
	query := bson.M{
		"userId":       userID,
		"providerType": "oauth",
		"providerName": "dexcom",
		"dataSetIds": bson.M{
			"$exists": true,
			"$not": bson.M{
				"$size": 0,
			},
		},
		"earliestDataTime": bson.M{
			"$exists": true,
		},
		"latestDataTime": bson.M{
			"$exists": true,
		},
	}

	dataSource := bson.M{}
	err := c.client.Database("tidepool").Collection("data_sources").FindOne(c.context, query).Decode(&dataSource)
	if err == mongo.ErrNoDocuments {
		return nil, nil
	}

	if err != nil {
		return nil, err
	}

	return dataSource, nil
}

// HasMedtronicLoopDataAfter checks the database to see if Loop data exists for `userID` that originated
// from a Medtronic device after `date`
func (t *TimeseriesStoreClient) HasMedtronicLoopDataAfter(userID string, date string) (bool, error) {
	if userID == "" {
		return false, errors.New("user id is missing")
	}
	if date == "" {
		return false, errors.New("date is missing")
	}

	query := bson.D{
		{Key: "_active", Value: true},
		{Key: "_userId", Value: userID},
		{Key: "time", Value: bson.D{{Key: "$gte", Value: date}}},
		{Key: "origin.payload.device.manufacturer", Value: "Medtronic"},
	}

	err := dataCollection(c).FindOne(c.context, query).Err()
	if err == mongo.ErrNoDocuments {
		return false, nil
	}

	return err == nil, err
}

// GetLoopableMedtronicDirectUploadIdsAfter returns all Upload IDs for `userID` where Loop data was found
// for a Medtronic device after `date`.
func (t *TimeseriesStoreClient) GetLoopableMedtronicDirectUploadIdsAfter(userID string, date string) ([]string, error) {
	if userID == "" {
		return nil, errors.New("user id is missing")
	}
	if date == "" {
		return nil, errors.New("date is missing")
	}

	model := new ([]models.Upload)
	query := t.db.Model(model)
	deviceModels := []string{"523", "523K", "554", "723", "723K", "754"}
	query = query.
		Where("user_id = ?", userID).
		Where("time > ", date).
		Where("deviceModel IN (?) ", pg.In(deviceModels))

	resultErr := query.Select()
	if resultErr != nil {
		return nil, resultErr
	}

	uploadIds := make([]string, len(*model))
	for index, object := range *model {
		uploadIds[index] = object.UploadId
	}

	return uploadIds, nil

}
 */

func (t *TimeseriesStoreClient) getModelType(modelType string) (interface{}, error) {
	var model interface{}
	switch modelType {
	case "physicalActivity":
		model = new ([]models.PhysicalActivity)
	case "basal":
		model = new ([]models.Basal)
	case "cbg":
		model = new ([]models.Cbg)
	case "smbg":
		model = new ([]models.Smbg)
	case "bloodKetone":
		return nil, errors.New("BloodKetone not yet implemented")
		//model = new ([]models.BloodKetone)
	case "bolus":
		model = new ([]models.Bolus)
	case "wizard":
		model = new ([]models.Wizard)
	case "deviceEvent":
		model = new ([]models.DeviceEvent)
	case "food":
		model = new ([]models.Food)
	case "insulin":
		//	model = new ([]models.Insulin)
		return nil,  errors.New("Insulin not yet implemented")
	case "cgmSettings":
		model = new ([]models.CgmSettings)
	case "pumpSettings":
		model = new ([]models.PumpSettings)
	case "reportedState":
		//model = new ([]models.ReportedState)
		return nil,  errors.New("Reported State not yet implemented")
	case "upload":
		model = new ([]models.Upload)
	default:
		return nil,errors.New(modelType + " not yet implemented")
	}

	return model, nil
}


// GetDeviceData returns all of the device data for a user
func (t *TimeseriesStoreClient) GetDeviceData(p *Params) (StorageIterator, error) {

	// _schemaVersion is still in the list of fields to remove. Although we don't query for it, data can still exist for it
	// until BACK-1281 is done.
	//removeFieldsForReturn := bson.M{"_id": 0, "_userId": 0, "_groupId": 0, "_version": 0, "_active": 0, "_schemaVersion": 0, "createdTime": 0, "modifiedTime": 0}

	latest := &tsIterator{pos: -1}

	var typeRanges []string
	if len(p.Types) > 0 && p.Types[0] != "" {
		typeRanges = p.Types
	} else {
		typeRanges = []string{"physicalActivity", "basal", "cbg", "smbg", "bloodKetone", "bolus", "wizard", "deviceEvent", "food", "insulin", "cgmSettings", "pumpSettings", "reportedState", "upload"}
	}

	fmt.Println("Type Ranges: ", typeRanges)

	var err error

	for _, theType := range typeRanges {

		count := 0
		fmt.Println("Type:", theType)
		model, err := t.getModelType(theType);
		if err != nil  || model == nil{
			continue
			//return nil, nil
		}

		query := t.db.Model(model)
		query = query.
			Where("user_id = ?", p.UserID)
			//Where("_active = ?", true).
			//Where("type = ?", theType)

		query = t.generateTimeseriesQuery(query, p).Order("time desc")
		if p.Latest {
			query = query.Limit(1)
		}
		resultErr := query.Select()
		if resultErr != nil {
			err = resultErr
			break
		}

		// Now we have to add all the elements we queried for to list
		v := reflect.ValueOf(model)
		if v.Kind() == reflect.Ptr {
			// get the value that the pointer v points to.
			items := v.Elem()
			if items.Kind() == reflect.Slice {
				for i := 0; i < items.Len(); i++ {
					item := items.Index(i)

					// Set type field since we do not store it
					v := item.FieldByName("Type")
					if v.IsValid() {
						v.SetString(theType)
					}
					latest.results = append(latest.results, item.Interface())
					count = count + 1
				}
			}
		}
		fmt.Println("Count:", count)
	}
	return latest, err

	//opts := options.Find().SetProjection(removeFieldsForReturn)
	//return dataCollection(c).
	//	Find(c.context, generateMongoQuery(p), opts)
}

// generateTimeseriesQuery takes in a number of parameters and constructs a mongo query
// to retrieve objects from the Tidepool database. It is used by the router.Add("GET", "/{userID}"
// endpoint, which implements the Tide-whisperer API. See that function for further documentation
// on parameters
func (t *TimeseriesStoreClient) generateTimeseriesQuery(q *orm.Query, p *Params) *orm.Query {


	if len(p.SubTypes) > 0 && p.SubTypes[0] != "" {
		q = q.WhereIn("sub_type", p.SubTypes)
	}

	if p.Date.Start != "" && p.Date.End != "" {
		q = q.Where("time > ?", p.Date.Start).Where("time < ?", p.Date.End)
	} else if p.Date.Start != "" {
		q = q.Where("time > ?", p.Date.Start)
	} else if p.Date.End != "" {
		q = q.Where("time < ?", p.Date.End)
	}

	if !p.Carelink {
		//q = q.Where("time != carelink")
	}

	if p.DeviceID != "" {
		q = q.Where("device_id = ?", p.DeviceID)
	}

	// If we have an explicit upload ID to filter by, we don't need or want to apply any further
	// data source-based filtering
	if p.UploadID != "" {
		q = q.Where("upload_id = ?", p.UploadID)
	} else {
		/*
		andQuery := []bson.M{}
		if !p.Dexcom && p.DexcomDataSource != nil {
			dexcomQuery := []bson.M{
				{"type": bson.M{"$ne": "cbg"}},
				{"uploadId": bson.M{"$in": p.DexcomDataSource["dataSetIds"]}},
			}
			earliestDataTime := p.DexcomDataSource["earliestDataTime"].(primitive.DateTime).Time().UTC()
			dexcomQuery = append(dexcomQuery, bson.M{"time": bson.M{"$lt": earliestDataTime.Format(time.RFC3339)}})
			latestDataTime := p.DexcomDataSource["latestDataTime"].(primitive.DateTime).Time().UTC()
			dexcomQuery = append(dexcomQuery, bson.M{"time": bson.M{"$gt": latestDataTime.Format(time.RFC3339)}})
			andQuery = append(andQuery, bson.M{"$or": dexcomQuery})
		}

		if !p.Medtronic && len(p.MedtronicUploadIds) > 0 {
			medtronicQuery := []bson.M{
				{"time": bson.M{"$lt": p.MedtronicDate}},
				{"type": bson.M{"$nin": []string{"basal", "bolus", "cbg"}}},
				{"uploadId": bson.M{"$nin": p.MedtronicUploadIds}},
			}
			andQuery = append(andQuery, bson.M{"$or": medtronicQuery})
		}

		if len(andQuery) > 0 {
			groupDataQuery["$and"] = andQuery
		}
		*/
	}

	return q
}

func (t *tsIterator) Next(context.Context) bool {
	t.pos++
	return t.pos < len(t.results)
}

func (t *tsIterator) Retrieve() (interface{}, error) {
	return t.results[t.pos], nil
}

func (t *tsIterator) Close(context.Context) error {
	return nil
}