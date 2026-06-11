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
		field.String("email").Optional(),
		field.String("password_hash").Sensitive(),
		field.String("display_name").Optional(),
		field.String("avatar_url").Optional(),
		field.String("oidc_issuer").Optional(),
		field.String("oidc_subject").Optional().Sensitive(),
		field.String("oidc_email").Optional(),
		field.Bool("is_admin").Default(false),
		field.Time("created_at"),
	}
}

func (User) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("tenant").Unique(),
		index.Fields("oidc_issuer", "oidc_subject").Unique(),
	}
}
