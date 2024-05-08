# Prompt Grader

This is a tool created for the purpose of rating the accuracy of provided prompts against the provided test sets

### Specs:

I want the ability to select which LLM's will be in use. (Maybe a checkbox with a required API key for each if it is decided to be in use).
This will also require the ability to select the engine per added LLM. (GPT-3.5, GPT-4, Claude Magnum Opus, etc.)

I want the ability to pass in arbitrary tests. These will be ran against the provided prompts and then used to calculate the percent accuracy.

### Rules:

Arbitary tests need to follow a standard. They require you to return true or false for each test case

OPTIONS:

Data set - iterates through a file you provide, A + B = C and then checks if C === C (T/F)
OR
Tests - Checks expected output from prompts and again returns (T/F)

OR both..
