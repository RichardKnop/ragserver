package genai

const ragTemplateStr = `
I will ask you a question and will provide some additional context information.
Assume the context information is factual and correct and do not consider any
other information outside of the context. Assume the question relates to a
specific company and the context is about that commpany.

If the question relates to the context, answer it using the context.
If the question does not relate to the context, simply return empty response.

For example, let's say the context has nothing in it about scope 1 emissions;
then if I ask you about scope 1 emissions, just return empty answer.

Answer the question according to provided schema. Schema defines a text.
The text field is a string. Text field should contain full answer to the question.


Question:
%s

Context:
%s
`

const ragTemplateMetricValue = `
I will ask you a question and will provide some additional context information.
Assume the context information is factual and correct and do not consider any
other information outside of the context. Assume the question relates to a
specific company and the context is about that commpany.

If the question relates to the context, answer it using the context.
If the question does not relate to the context, simply return empty response.

For example, let's say the context has nothing in it about scope 1 emissions;
then if I ask you about scope 1 emissions, just return empty answer.

Answer the question according to provided schema. Schema defines a text field
and a metric field. The text field is a string. The metric field is an object 
that has a value and a unit. The value is a number and the unit is a string.
Text field should contain full answer to the question. Metric field should 
contain structured answer with a numeric value and a unit of measurement.

Question:
%s

Context:
%s
`
