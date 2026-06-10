package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
)

type User struct {
	ent.Schema
}

func (User) Fields() []ent.Field {
	return []ent.Field{
		field.String("tenant"),
		field.String("username").Unique(),
		field.String("password_hash"),
		field.String("display_name").Optional(),
		field.String("avatar_url").Optional(),
		field.Bool("is_admin").Default(false),
		field.Time("created_at"),
	}
}

func (User) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("tenant").Unique(),
	}
}
