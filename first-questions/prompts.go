package prompts

import "fmt"

type Summary struct {
	Prompt         string
	entitiesSource string
}

func NewSummary(entitiesSource string) *Summary {
	return &Summary{
		entitiesSource: entitiesSource,
	}
}

func (s *Summary) Generate() (string, error) {
	var tableJson = `{
		"tables": [
		  {
			"name": "TableName",
			"fields": [
			  {
				"name": "FieldName",
				"type": "DataType",
				"isPrimaryKey": boolean,
				"isForeignKey": boolean,
				"references": "ReferencedTableName" (if applicable)
			  }
			]
		  }
		],
		"relationships": [
		  {
			"from": "TableName1",
			"fromField": "FieldName1",
			"to": "TableName2",
			"toField": "FieldName2",
			"type": "OneToMany/ManyToOne/etc"
		  }
		]
	  }`

	var prompt = `Analyze the following text representation of an ER diagram and convert it into a JSON data structure.

	The input format is as follows:

	- Tables are defined with \"TABLE\" followed by the table name
	- Fields are listed under each table with format: field_name (data_type, constraints)
	- Relationships are defined with \"RELATIONSHIP\" followed by the connection details

	When data types are not explicitly provided, predict the most likely data type based on the field name and common database practices. For example:
	- Fields like 'id', 'ID', or ending with '_id' are likely to be INTEGER
	- Fields with 'name', 'title', or 'description' are likely to be VARCHAR or TEXT
	- Fields with 'date' or ending with '_at' (like 'created_at') are likely to be DATE or DATETIME
	- Fields like 'is_' or 'has_' are likely to be BOOLEAN
	- Numeric fields (like 'price', 'quantity', 'amount') could be INTEGER, DECIMAL, or FLOAT depending on context

	Respond with a JSON object that has two main keys: "explanation" for any prose explanation, and "data" for the structured data output. The "data" should contain the parsed ER diagram information.

	Your response should be valid JSON and should follow this structure:

	{
	"explanation": "String containing any explanatory text or comments about the conversion process",
	"data": ` + tableJson + `
	}`

	prompt = fmt.Sprintf("%q", prompt)
	// remove the quotes... TODO: sucks
	prompt = prompt[1 : len(prompt)-1]

	var fullPrompt = fmt.Sprintf("%s %s", prompt, s.entitiesSource)

	return fullPrompt, nil
}
