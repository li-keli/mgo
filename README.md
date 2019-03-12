[![Build Status](https://travis-ci.org/globalsign/mgo.svg?branch=master)](https://travis-ci.org/globalsign/mgo) [![GoDoc](https://godoc.org/github.com/li-keli/mgo?status.svg)](https://godoc.org/github.com/li-keli/mgo)

The MongoDB driver for Go
-------------------------

Add support for some mongodb 4.0 features, but only at an experimental stage.

add func:

```go
// UpdateWithArrayFilters allows passing an array of filter documents that determines
// which array elements to modify for an update operation on an array field. The multi parameter
// determines whether the update should update multiple documents (true) or only one document (false).
// 
// See example: https://docs.mongodb.com/manual/reference/method/db.collection.update/#update-arrayfiltersi
// 
// Note this method is only compatible with MongoDB 3.6+.
func (c *Collection) UpdateWithArrayFilters(selector, update, arrayFilters interface{}, multi bool) (*ChangeInfo, error)
```

[GoDoc](https://godoc.org/github.com/li-keli/mgo).

A [sub-package](https://godoc.org/github.com/li-keli/mgo/bson) that implements the [BSON](http://bsonspec.org) specification is also included, and may be used independently of the driver.

## Supported Versions

`mgo` is known to work well on (and has integration tests against) MongoDB v3.0, 3.2, 3.4, 3.6 and 4.0 . 
