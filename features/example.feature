Feature: When called, the Lambda will hit the external API, fetch the message and write it to various destinations

  Scenario: Fetch the message from the external API and write it to various destinations
    Given the Lambda is triggered
    Then the external API endpoint is hit
    And the Lambda writes the message to the log
    And the Lambda writes a message to the SQS queue
    And the Lambda sends a notification to the SNS topic
    And the Lambda writes the message to DynamoDB
    And the Lambda writes the message to the Aurora Postgres database