package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/schema/field"
)

type SystemConfig struct {
	ent.Schema
}

func (SystemConfig) Fields() []ent.Field {
	return []ent.Field{
		field.String("key").Unique(),
		field.String("payload"),
		field.Time("updated_at"),
	}
}
