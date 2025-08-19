package ragserver

const ragTemplateStr = `
I will ask you a question and will provide some additional context information.
Assume the context information is factual and correct and do not consider any
other information outside of the context. Assume the question relates to a
specific company and the context is about that commpany.

If the question relates to the context, answer it using the context.
If the question does not relate to the context, simply return empty string as answer.

For example, let's say the context has nothing in it about scope 1 emissions;
then if I ask you about scope 1 emissions, just answer with empty string.

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
If the question does not relate to the context, simply return empty string as answer.

For example, let's say the context has nothing in it about scope 1 emissions;
then if I ask you about scope 1 emissions, just answer with empty string.

On the first line of your answer, include a sentence or a phrase that contains 
the answer to the question. For example "Scope 1 is 77,476".
On new line, return the answer as a valid JSON number. For example 42 or 3.14. 
Do not wrap it in quotes, do not add any additional text. 

Question:
%s

Context:
%s
`
