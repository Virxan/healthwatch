Feature: Health check of a single target
  As an operator
  I want Healthwatch to correctly classify a target as up or down
  So that I can trust the dashboard and the API

  Scenario: A site responding with 200 is reported as up
    Given a target website that responds with status 200
    When Healthwatch checks the target
    Then the result status should be "up"
    And the result should record a non-negative latency

  Scenario: A site responding with a server error is reported as down
    Given a target website that responds with status 500
    When Healthwatch checks the target
    Then the result status should be "down"
    And the result should have an error message

  Scenario: An unreachable site is reported as down
    Given a target website that is unreachable
    When Healthwatch checks the target
    Then the result status should be "down"
    And the result should have an error message

  Scenario: An HTTPS target with a valid certificate reports days remaining
    Given a target website served over HTTPS with a valid certificate
    When Healthwatch checks the target
    Then the result status should be "up"
    And the result should report a positive number of TLS days remaining
