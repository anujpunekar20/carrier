package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/schema/field"
)

// Job holds the schema definition for the Job entity.
type Job struct {
	ent.Schema
}

// Fields of the Job.
func (Job) Fields() []ent.Field {
	return []ent.Field{
		field.String("title"),
		field.String("company"),
		field.String("location").Optional(),
		field.String("salary").Optional(),
		field.String("employment_type").Optional(),
		field.String("url").Unique(),
		field.String("source"),
		field.Text("description").Optional(),
		field.Time("posted_on").Optional(),
		field.Time("scraped_at"),
	}
}
