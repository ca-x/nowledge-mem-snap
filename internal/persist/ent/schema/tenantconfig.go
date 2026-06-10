package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/schema/field"
)

type TenantConfig struct {
	ent.Schema
}

func (TenantConfig) Fields() []ent.Field {
	return []ent.Field{
		field.String("tenant").Unique(),
		field.String("payload").Sensitive(),
		field.Time("updated_at"),
	}
}
