/*
Package null implements nullable sql fields that are safely marshalable from/to json and text.

Each of the types defined within implement the `Stringer`, `sql.Scanner`, `json.Marshaller`,
`json.Unmarshaller`, `encoding.TextMarshaller`, and `encoding.TextUnmarshaller`` interfaces.  This makes
them safe to use struct fields that can be saved and read from a database as well as from http json
requests and parsed from string values.  They can also be safely printed in a `fmt.Printf``

Where possible we try to alias and delegate known and well tested types, such as pd.NullTime
and uuid.NullUUID.  These types are provide more complete interfaces for our expected normal usage
in Labs.
*/
package null
