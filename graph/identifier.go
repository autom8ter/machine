package graph

import (
	"encoding/hex"
	"fmt"
	"math/rand"
)

type Identifier interface {
	// ID returns a string id
	ID() string
	// Type returns the string type
	Type() string
	// String returns a concatenation of id and type
	String() string
}

type identity struct {
	id  string
	typ string
}

func (i *identity) ID() string {
	return i.id
}

func (i *identity) Type() string {
	return i.typ
}

func (i *identity) String() string {
	return fmt.Sprintf("%s.%s", i.typ, i.id)
}

// NewIdentifier returns a new Identifier implementation. If an id is not specified, a random one will be generated automatically.
func NewIdentifier(typ string, id string) Identifier {
	if id == "" {
		id = genID()
	}
	return &identity{
		id:  id,
		typ: typ,
	}
}

func genID() string {
	b := make([]byte, 16)
	_, err := rand.Read(b)
	if err != nil {
		return ""
	}
	return hex.EncodeToString(b)
}
