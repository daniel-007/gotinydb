package gotinydb

import (
	"bytes"
	"os"
	"reflect"
	"testing"

	"github.com/blevesearch/bleve"
)

var (
	testDB  *DB
	testCol *Collection

	testPath      = os.TempDir() + "/testDB"
	testConfigKey = [32]byte{}

	testColName = "collection name"

	testUserID = "test ID"
	testUser   = &testUserStruct{
		"toto",
		"userName@internet.org",
		&Account{"Github", "https://www.github.com"},
	}

	testIndexName = "email"
	// testIndexSelector         = "Email"
	testIndexNameAccounts = "accounts"
	// testIndexSelectorAccounts = []string{"accounts", "Name"}
)

type (
	testUserStruct struct {
		Name  string   `json:"name"`
		Email string   `json:"email"`
		Oauth *Account `json:"oauth"`
	}
	Account struct {
		Name, URL string
	}
)

func (t *testUserStruct) Type() string {
	return "local.testUserStruct"
}

func init() {
	os.RemoveAll(testPath)
}

func TestMain(t *testing.T) {
	defer clean()
	err := open(t)
	if err != nil {
		return
	}

	retrievedUser := new(testUserStruct)
	_, err = testCol.Get(testUserID, retrievedUser)
	if err != nil {
		t.Error(err)
		return
	}

	if !reflect.DeepEqual(retrievedUser, testUser) {
		t.Errorf("the users are not equal. Put %v and get %v", testUser, retrievedUser)
		return
	}

	query := bleve.NewQueryStringQuery(testUser.Email)
	searchRequest := bleve.NewSearchRequestOptions(query, 10, 0, true)
	var searchResult *SearchResult
	searchResult, err = testCol.Search(testIndexName, searchRequest)
	if err != nil {
		t.Error(err)
		return
	}

	if testing.Verbose() {
		t.Log("searchResult", searchResult)
	}

	query = bleve.NewQueryStringQuery(testUser.Name)
	searchRequest = bleve.NewSearchRequestOptions(query, 10, 0, true)
	searchResult, err = testCol.Search(testIndexName, searchRequest)
	if err == nil {
		t.Errorf("the index must return no result but had %s", searchResult.BleveSearchResult.String())
		return
	}

	if testing.Verbose() {
		t.Log("searchResult", searchResult)
	}

	err = testDB.Close()
	if err != nil {
		t.Error(err)
		return
	}

	testDB = nil
	testCol = nil

	testDB, err = Open(testPath, testConfigKey)
	if err != nil {
		t.Error(err)
		return
	}

	testCol, err = testDB.Use(testColName)
	if err != nil {
		t.Error(err)
		return
	}

	query = bleve.NewQueryStringQuery(testUser.Email)
	searchRequest = bleve.NewSearchRequestOptions(query, 10, 0, true)
	searchResult, err = testCol.Search(testIndexName, searchRequest)
	if err != nil {
		t.Error(err)
		return
	}

	if testing.Verbose() {
		t.Log("searchResult", searchResult)
	}

	err = testCol.Delete(testUserID)
	if err != nil {
		t.Error(err)
		return
	}

	query = bleve.NewQueryStringQuery(testUser.Email)
	searchRequest = bleve.NewSearchRequestOptions(query, 10, 0, true)
	searchResult, err = testCol.Search(testIndexName, searchRequest)
	if err == nil {
		t.Errorf("the index should returns no result but had %s", searchResult.BleveSearchResult.String())
		return
	}
}

func open(t *testing.T) (err error) {
	testDB, err = Open(testPath, testConfigKey)
	if err != nil {
		t.Error(err)
		return err
	}

	testCol, err = testDB.Use(testColName)
	if err != nil {
		t.Error(err)
		return err
	}

	userDocumentMapping := bleve.NewDocumentStaticMapping()

	emailFieldMapping := bleve.NewTextFieldMapping()
	userDocumentMapping.AddFieldMappingsAt("email", emailFieldMapping)


	indexMapping := bleve.NewIndexMapping()
	indexMapping.AddDocumentMapping("local.testUserStruct", userDocumentMapping)

	err = testCol.SetBleveIndex(testIndexName, indexMapping)
	if err != nil {
		t.Error(err)
		return err
	}

	// userMapping := bleve.NewDocumentMapping()

	// nameFieldMapping := bleve.NewTextFieldMapping()
	// nameFieldMapping.Analyzer = "en"
	// userMapping.AddFieldMappingsAt("Name", nameFieldMapping)
	// userMapping.AddFieldMappingsAt("Email", nameFieldMapping)

	// accountMapping := bleve.NewDocumentMapping()

	// indexMapping := bleve.NewIndexMapping()
	// indexMapping.AddDocumentMapping("testUserStruct", userMapping)
	// indexMapping.AddDocumentMapping("Account", accountMapping)

	// err = testCol.SetBleveIndex(testIndexNameAccounts, indexMapping)
	// if err != nil {
	// 	t.Error(err)
	// 	return err
	// }

	err = testCol.Put(testUserID, testUser)
	if err != nil {
		t.Error(err)
		return err
	}

	return
}

func clean() {
	testDB.Close()
	os.RemoveAll(testPath)
}

func TestBackup(t *testing.T) {
	defer clean()
	err := open(t)
	if err != nil {
		return
	}

	var backup bytes.Buffer

	err = testDB.Backup(&backup)
	if err != nil {
		t.Error(err)
		return
	}

	restoredDBPath := os.TempDir() + "/restoredDB"
	defer os.RemoveAll(restoredDBPath)

	var restoredDB *DB
	restoredDB, err = Open(restoredDBPath, testConfigKey)
	if err != nil {
		t.Error(err)
		return
	}

	err = restoredDB.Load(&backup)
	if err != nil {
		t.Error(err)
		return
	}

	var col2 *Collection
	col2, err = restoredDB.Use(testColName)
	if err != nil {
		t.Error(err)
		return
	}

	query := bleve.NewQueryStringQuery(testUser.Email)
	searchRequest := bleve.NewSearchRequestOptions(query, 10, 0, true)
	var searchResult *SearchResult
	searchResult, err = col2.Search(testIndexName, searchRequest)
	if err != nil {
		t.Error(err)
		return
	}

	if testing.Verbose() {
		t.Log("searchResult", searchResult)
	}
}
