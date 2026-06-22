Feature: Healthwatch API contract
  As an operator
  I want the deployed Healthwatch API to honour its contract
  So that dashboards and automation built on top of it never break silently

  Background:
    Given the Healthwatch API base URL defaults to "http://localhost:8080"

  Scenario: The liveness endpoint reports healthy
    When I GET "/healthz"
    Then the response status should be 200

  Scenario: Listing checks returns a JSON array
    When I GET "/api/v1/checks"
    Then the response status should be 200
    And the response should be a JSON array
    And the response content type should be "application/json"

  Scenario: Requesting an unknown target returns 404
    When I GET "/api/v1/checks/this-target-does-not-exist"
    Then the response status should be 404

  Scenario: The dashboard is served as HTML
    When I GET "/"
    Then the response status should be 200
    And the response content type should be "text/html; charset=utf-8"
