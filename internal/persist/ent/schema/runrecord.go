package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
)

type RunRecord struct {
	ent.Schema
}

func (RunRecord) Fields() []ent.Field {
	return []ent.Field{
		field.String("tenant"),
		field.String("run_id"),
		field.String("payload"),
		field.Time("started_at"),
	}
}

func (RunRecord) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("tenant", "run_id").Unique(),
		index.Fields("tenant", "started_at"),
	}
}
