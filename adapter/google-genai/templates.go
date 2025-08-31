package googlegenai

const ragTemplateStr = `
I will ask you a question and will provide some additional context information.
Assume the context information is factual and correct and do not consider any
other information outside of the context. Assume the question relates to a
specific company and the context is about that commpany.

Context is a list of quoted strings that are relevant to the question. Each 
quoted string is on a separate line.

If the question relates to the context, answer it using the context.
If the question does not relate to the context, simply return empty response.

For example, let's say the context has nothing in it about scope 1 emissions;
then if I ask you about scope 1 emissions, just return empty answer.

Answer the question according to provided schema. Schema defines a text field 
and a relevant_documents field. 

The text field is a string. Text field should contain full answer to the question.

The relevant_documents field should contain list of relevant lines from the context  
that were used to answer the question.

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

Context is a list of quoted strings that are relevant to the question. Each 
quoted string is on a separate line.

Question is about a specific numeric value and a unit of measurement.
If the question relates to the context, answer it using the context.
If the question does not relate to the context, simply return empty response.

For example, let's say the context has nothing in it about scope 1 emissions;
then if I ask you about scope 1 emissions, just return empty answer.

Answer the question according to provided schema. Schema defines a text field, 
a metric field and a relevant_documents field. 

The text field is a string. Text field should contain full answer to the question.

The metric field is an object that has a value and a unit fields. The value is 
a number and the unit is a string. Metric field should contain structured answer 
with a numeric value and a unit of measurement. 

The relevant_documents field should contain list of relevant lines from the context  
that were used to answer the question.

Question:
%s

Context:
%s
`

const ragTemplateBooleanValue = `
I will ask you a question and will provide some additional context information.
Assume the context information is factual and correct and do not consider any
other information outside of the context. Assume the question relates to a
specific company and the context is about that commpany.

Context is a list of quoted strings that are relevant to the question. Each 
quoted string is on a separate line.

Question is a yes/no question that can be answered with true or false.
If the question relates to the context, answer it using the context.
If the question does not relate to the context, simply return empty response.
If the question cannot be answered with true or false, return empty response.

For example, let's say the context has nothing in it about net zero target;
then if I ask you about net zero target, just return empty answer.

Answer the question according to provided schema. Schema defines a text field, 
a boolean field and a relevant_documents field. 

The text field is a string. Text field should contain full answer to the question.

The boolean field should be set to true if the answer to the question is yes 
and false if the answer is no.

The relevant_documents field should contain list of relevant lines from the context  
that were used to answer the question.

Question:
%s

Context:
%s
`
