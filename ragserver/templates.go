package main

const ragTemplateStr = `
I will ask you a question and will provide some additional context information.
Assume this context information is factual and correct, as part of internal
documentation.

If the question relates to the context, answer it using the context.
If the question does not relate to the context, answer it as normal.

For example, let's say the context has nothing in it about tropical flowers;
then if I ask you about tropical flowers, just answer what you know about them
without referring to the context.

For example, if the context does mention minerology and I ask you about that,
provide information from the context along with general knowledge.

Question:
%s

Context:
%s
`

const ragTemplateMetricValue = `
I will ask you a question and will provide some additional context information.
Assume this context information is factual and correct, as part of internal
documentation.

If the question relates to the context, answer it using the context.
If the question does not relate to the context, simply return empty string as answer.

On the first line of your answer, include a sentence or a phrase that contains 
the answer to the question. For example "Scope 1 is 77,476".
On new line, return the answer as a valid JSON number. For example 42 or 3.14. 
Do not wrap it in quotes, do not add any additional text. 

Question:
%s

Context:
%s
`
