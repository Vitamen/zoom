package test

import (
	"github.com/stephenalexbrowne/zoom"
	"github.com/stephenalexbrowne/zoom/redis"
	"github.com/stephenalexbrowne/zoom/support"
	"github.com/stephenalexbrowne/zoom/util"
	"reflect"
	"testing"
)

func TestSave(t *testing.T) {
	support.SetUp()
	defer support.TearDown()

	// create and save a new person
	p := &support.Person{Name: "Bob", Age: 25}
	err := zoom.Save(p)
	if err != nil {
		t.Error(err)
	}

	// make sure it was assigned an id
	if p.Id == "" {
		t.Error("model was not assigned an id")
	}

	// get a connection
	conn := zoom.GetConn()
	defer conn.Close()

	// make sure the key exists
	key := "person:" + p.Id
	exists, err := zoom.KeyExists(key, conn)
	if err != nil {
		t.Error(err)
	}
	if exists == false {
		t.Error("model was not saved in redis")
	}

	// make sure the values are correct for the person
	results, err := redis.Strings(conn.Do("HMGET", key, "Name", "Age"))
	if err != nil {
		t.Error(err)
	}
	if results[0] != "Bob" {
		t.Errorf("Name of saved model was incorrect. Expected: %s. Got: %s.\n", "Bob", results[0])
	}
	if results[1] != "25" {
		t.Errorf("Age of saved model was incorrect. Expected: %s. Got: %s.\n", "25", results[1])
	}

	// make sure it was added to the index
	indexed, err := zoom.SetContains("person:index", p.Id, conn)
	if err != nil {
		t.Error(err)
	}
	if indexed == false {
		t.Error("model was not added to person:index")
	}
}

func TestFindById(t *testing.T) {
	support.SetUp()
	defer support.TearDown()

	// create and save a new model
	p := &support.Person{Name: "Bob", Age: 25}
	zoom.Save(p)

	// find the model using FindById
	result, err := zoom.FindById("person", p.Id).Exec()
	if err != nil {
		t.Error(err)
	}
	if result == nil {
		t.Error("result of FindById was nil")
	}
	pCopy, ok := result.(*support.Person)
	if !ok {
		t.Error("could not type assert result to *Person")
	}

	// make sure the found model is the same as original
	if pCopy.Id != p.Id {
		t.Errorf("Id was incorrect. Expected: %s. Got: %s.\n", p.Id, pCopy.Id)
	}
	if pCopy.Name != p.Name {
		t.Errorf("Name was incorrect. Expected: %s. Got: %s.\n", p.Name, pCopy.Name)
	}
	if pCopy.Age != p.Age {
		t.Errorf("Age was incorrect. Expected: %d. Got: %d.\n", p.Age, pCopy.Age)
	}
}

func TestScanById(t *testing.T) {
	support.SetUp()
	defer support.TearDown()

	// create and save a new model
	p := &support.Person{Name: "Bob", Age: 25}
	zoom.Save(p)

	// create a new model for to scan into
	pCopy := &support.Person{}

	// find the model using ScanById
	_, err := zoom.ScanById(pCopy, p.Id).Exec()
	if err != nil {
		t.Error(err)
	}

	// make sure the found model is the same as original
	if pCopy.Id != p.Id {
		t.Errorf("Id was incorrect. Expected: %s. Got: %s.\n", p.Id, pCopy.Id)
	}
	if pCopy.Name != p.Name {
		t.Errorf("Name was incorrect. Expected: %s. Got: %s.\n", p.Name, pCopy.Name)
	}
	if pCopy.Age != p.Age {
		t.Errorf("Age was incorrect. Expected: %d. Got: %d.\n", p.Age, pCopy.Age)
	}
}

func TestDelete(t *testing.T) {
	support.SetUp()
	defer support.TearDown()

	// create and save a new model
	p := &support.Person{Name: "Bob", Age: 25}
	zoom.Save(p)

	// delete it
	zoom.Delete(p)

	// get a connection
	conn := zoom.GetConn()
	defer conn.Close()

	// Make sure it's gone
	key := "person:" + p.Id
	exists, err := zoom.KeyExists(key, conn)
	if err != nil {
		t.Error(err)
	}
	if exists {
		t.Error("model key still exists")
	}

	// Make sure it was removed from index
	indexed, err := zoom.SetContains("person:index", p.Id, conn)
	if err != nil {
		t.Error(err)
	}
	if indexed {
		t.Error("model id is still in person:index")
	}
}

func TestDeleteById(t *testing.T) {
	support.SetUp()
	defer support.TearDown()

	// create and save a new model
	p := &support.Person{Name: "Bob", Age: 25}
	zoom.Save(p)

	// delete it
	zoom.DeleteById("person", p.Id)

	// get a connection
	conn := zoom.GetConn()
	defer conn.Close()

	// Make sure it's gone
	key := "person:" + p.Id
	exists, err := zoom.KeyExists(key, conn)
	if err != nil {
		t.Error(err)
	}
	if exists {
		t.Error("model key still exists")
	}

	// Make sure it was removed from index
	indexed, err := zoom.SetContains("person:index", p.Id, conn)
	if err != nil {
		t.Error(err)
	}
	if indexed {
		t.Error("model id is still in person:index")
	}
}

func TestFindAll(t *testing.T) {
	support.SetUp()
	defer support.TearDown()

	// Create and save some Pets
	// we can assume this works since
	// Save() was tested previously
	p1 := &support.Person{Name: "Elroy", Age: 26}
	p2 := &support.Person{Name: "Fred", Age: 27}
	p3 := &support.Person{Name: "Gus", Age: 28}
	zoom.Save(p1)
	zoom.Save(p2)
	zoom.Save(p3)

	// query to get a list of all the pets
	results, err := zoom.FindAll("person")
	if err != nil {
		t.Error(err)
	}

	// make sure results is the right length
	if len(results) != 3 {
		t.Errorf("results was not the right length. Expected: %d. Got: %d.\n", 3, len(results))
	}

	// make sure each item in results is correct
	// NOTE: this is tricky because the order can
	// change in redis and we haven't asked for any sorting
	expectedNames := []string{p1.Name, p2.Name, p3.Name}
	gotNames := []string{}
	for i, result := range results {
		// each item in results should be able to be casted to *Person
		pResult, ok := result.(*support.Person)
		if !ok {
			t.Errorf("Couldn't cast results[%d] to *Person", i)
		}
		// each item in results should have a valid Id
		if pResult.Id == "" {
			t.Error("Id was not set for Person: ", pResult)
		}
		gotNames = append(gotNames, pResult.Name)
	}

	equal, msg := util.CompareAsStringSet(expectedNames, gotNames)
	if !equal {
		t.Errorf("\nexpected: %v\ngot: %v\nmsg: %s\n", expectedNames, gotNames, msg)
	}
}

func TestSaveWithList(t *testing.T) {
	support.SetUp()
	defer support.TearDown()

	// create a modelWithList model
	m := &support.ModelWithList{
		List: []string{"one", "two", "three"},
	}

	// save it
	if err := zoom.Save(m); err != nil {
		t.Error(err)
	}

	// get a connection
	conn := zoom.GetConn()
	defer conn.Close()

	// make sure the list was saved properly
	listKey := "modelWithList:" + m.Id + ":List"
	listCopy, err := redis.Strings(conn.Do("LRANGE", listKey, 0, -1))
	if err != nil {
		t.Error(err)
	}
	if !reflect.DeepEqual(m.List, listCopy) {
		t.Errorf("List was not the same.\nExpected: %v\nGot: %v\n", m.List, listCopy)
	}
}

func TestFindByIdWithList(t *testing.T) {
	support.SetUp()
	defer support.TearDown()

	// create and save a modelWithList model
	m := &support.ModelWithList{
		List: []string{"one", "two", "three"},
	}
	zoom.Save(m)

	// retrieve using FindById
	mCopy := &support.ModelWithList{}
	if _, err := zoom.ScanById(mCopy, m.Id).Exec(); err != nil {
		t.Error(err)
	}

	// make sure the list is the same
	if !reflect.DeepEqual(m.List, mCopy.List) {
		t.Errorf("List was not the same.\nExpected: %v\nGot: %v\n", m.List, mCopy.List)
	}
}

func TestSaveWithSet(t *testing.T) {
	support.SetUp()
	defer support.TearDown()

	// create a modelWithSet model
	m := &support.ModelWithSet{
		Set: []string{"one", "two", "three", "three"},
	}

	// save it
	if err := zoom.Save(m); err != nil {
		t.Error(err)
	}

	// get a connection
	conn := zoom.GetConn()
	defer conn.Close()

	// make sure the set was saved properly
	setKey := "modelWithSet:" + m.Id + ":Set"
	setCopy, err := redis.Strings(conn.Do("SMEMBERS", setKey))
	if err != nil {
		t.Error(err)
	}
	set := []string{"one", "two", "three"}
	equal, msg := util.CompareAsStringSet(set, setCopy)
	if !equal {
		t.Error(msg)
	}
}

func TestFindByIdWithSet(t *testing.T) {
	support.SetUp()
	defer support.TearDown()

	// create and save a modelWithSet model
	m := &support.ModelWithSet{
		Set: []string{"one", "two", "three", "three"},
	}
	zoom.Save(m)

	// retrieve using FindById
	mCopy := &support.ModelWithSet{}
	if _, err := zoom.ScanById(mCopy, m.Id).Exec(); err != nil {
		t.Error(err)
	}

	// make sure the set is what we expect
	set := []string{"one", "two", "three"}
	equal, msg := util.CompareAsStringSet(set, mCopy.Set)
	if !equal {
		t.Error(msg)
	}
}
