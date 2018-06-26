# GoTinyDB

[![GoDoc](https://godoc.org/github.com/alexandrestein/gotinydb?status.svg)](https://godoc.org/github.com/alexandrestein/gotinydb) [![Build Status](https://travis-ci.org/alexandrestein/gotinydb.svg?branch=master)](https://travis-ci.org/alexandrestein/gotinydb) [![codecov](https://codecov.io/gh/alexandreStein/GoTinyDB/branch/master/graph/badge.svg)](https://codecov.io/gh/alexandreStein/GoTinyDB) [![Go Report Card](https://goreportcard.com/badge/github.com/alexandrestein/gotinydb)](https://goreportcard.com/report/github.com/alexandrestein/gotinydb) [![License](https://img.shields.io/badge/License-Apache--2.0-blue.svg)](http://www.apache.org/licenses/LICENSE-2.0)

The goal is to have a fairly simple database which is light and don't needs to fit in RAM. It supports indexing for most of the basic Golang types.

## Installing

```bash
go get -u github.com/alexandrestein/gotinydb
```

## Getting started

The package is supposed to be used inside your software and at this point it is not suppose to be a "real" database service.

### Open database

```golang
db, initErr := New(internalTesting.Path)
if initErr != nil {
  log.Fatal(initErr.Error())
  return
}
defer db.Close()
```

### Open collection

```golang
col, colErr := db.Use("collectionName")
if colErr != nil {
  log.Fatal("opening test collection: %s", colErr.Error())
  return
}
```

### Setup an index for future queries

```golang
// If you have user object like this:
// {UserName: string, Address: {Street: string, Num: int, City: string, ZIP: int}}
// and you want to index the username and the ZIP code.
index := NewIndex("userName", []string{"UserName"},  vars.StringIndex)
if err := c.SetIndex(index); err != nil {
  log.Fatal(err)
}
index := NewIndex("zip", []string{"Address","ZIP"},  vars.IntIndex)
if err := c.SetIndex(index); err != nil {
  log.Fatal(err)
}
```

There is many types of index. Take a look at the [index documentation](https://godoc.org/github.com/alexandrestein/gotinydb/index).

### Put some data in the collection

```golang
putErr := col.Put(objectID, objectOrBytes)
if putErr != nil {
  log.Error(putErr)
  return
}
```

The content can be an object or a stream of bytes. If it's a stream it needs to
have the form of `[]byte{}`.
This will adds and updates existing values.

### Get some data from the collection directly by it's the ID

```golang
getErr := col.Get(objectID, receiver)
if getErr != nil {
  t.Error(getErr)
  return
}
```

The receiver can be an object pointer or a stream of bytes. If it's a stream it needs to
have the form of `*bytes.Buffer`.

### Get objects by query

```golang
// Get IDs of object with ZIP code greater than 50 limited to 5 responses ordered via zip code
q := NewQuery().SetOrder([]string{"Address", "ZipCode"}, true).Get(
  NewFilter(Greater).SetSelector([]string{"Address", "ZipCode"}).EqualWanted().
    CompareTo(uint(50)),
).SetLimits(5, 0)

// Do the query
response, err := c.Query(q)
if err != nil {
  log.Fatal(err)
}

// Get the results
users := make([]*User, response.Len())
for i, _, v := gotResponse.First(); i >= 0; i, _, v = gotResponse.Next() {
  user := new(User)
  err := json.Unmarshal(v, user)
  if err != nil {
    t.Error(err)
    return
  }

  users[i] = user
}

```

This returns only a list of IDs. It's up to the caller to get the values he want
with the Get function.

## Built With

* [Badger](https://github.com/dgraph-io/badger) - Is the main storage engine
* [Bolt](https://github.com/boltdb/bolt) - Is the index engine
* [Structs](https://github.com/fatih/structs) - Used to cut objects in part for indexing

## To Do

* Background indexing
* Collection and Index deletion
* Add some tests

## Contributing

Please read [CONTRIBUTING.md](CONTRIBUTING.md) for details on our code of conduct, and the process for submitting pull requests to us.

## Vendoring

We use [dep](https://github.com/golang/dep) or [vgo](https://github.com/golang/vgo/) for vendoring.

## Versioning

The package is under heavy developement for now and is just for testing and developement at this point.
Ones the design will be finalised the version will start at `1.0.0`.
For futur the versions, see the [tags on this repository](https://github.com/alexandrestein/gotinydb/tags).

## Authors

* **Alexandre Stein** - [GitHub](https://github.com/alexandrestein)

<!-- See also the list of [contributors](https://github.com/your/project/contributors) who participated in this project. -->

## License

This project is licensed under the "Apache License, Version 2.0" - see the [LICENSE](LICENSE) file for details or follow this [link](http://www.apache.org/licenses/LICENSE-2.0).

## Acknowledgments

* I was looking for pure `golang` database for reasonable (not to big) data size. I checked [Tiedot](https://github.com/HouzuoGuo/tiedot) long time ago but the index is only for exact match which was not what I was looking for.
* B-Tree is a good way to have ordered elements and is extremely scalable.