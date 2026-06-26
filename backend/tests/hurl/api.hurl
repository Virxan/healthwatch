# Run with: hurl --variable base_url=http://localhost:8080 backend/tests/hurl/api.hurl
# (base_url defaults below if not passed on the command line)

GET {{base_url}}/health
HTTP 200
[Asserts]
jsonpath "$.status" == "ok"


GET {{base_url}}/items
HTTP 200
[Asserts]
jsonpath "$" exists


POST {{base_url}}/items
Content-Type: application/json
{
  "name": "hurl test item",
  "url": "https://example.com"
}
HTTP 201
[Captures]
created_id: jsonpath "$.id"
[Asserts]
jsonpath "$.name" == "hurl test item"
jsonpath "$.url" == "https://example.com"
jsonpath "$.id" isInteger
jsonpath "$.created_at" exists


GET {{base_url}}/items
HTTP 200
[Asserts]
jsonpath "$[*].id" contains {{created_id}}


POST {{base_url}}/items
Content-Type: application/json
{
  "name": "   ",
  "url": "https://example.com"
}
HTTP 400


POST {{base_url}}/items
Content-Type: application/json
{
  "name": "no url"
}
HTTP 400


POST {{base_url}}/items
Content-Type: application/json
{
  "name": "bad url",
  "url": "not-a-url"
}
HTTP 400
