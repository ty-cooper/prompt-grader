# Prompt Grader

This is a tool created for the purpose of rating the accuracy of provided prompts against the provided test sets

### Specs:

I want the ability to select which LLM's will be in use. (Maybe a checkbox with a required API key for each if it is decided to be in use).
This will also require the ability to select the engine per added LLM. (GPT-3.5, GPT-4, Claude Magnum Opus, etc.)

I want the ability to pass in arbitrary tests. These will be ran against the provided prompts and then used to calculate the percent accuracy.

An automatic report will then be generated from the test results and the provided prompts/engines/LLM's used. Similar to nmap html output.

-t --tests : Specify the tests location ... figure out what format will be needed
-l --llms : Specify which llms will be used in this run (-l gpt4, claude)
-p --prompt : Specify the prompt that will be used to run the test
-f --prompt-file : Read the prompt from a specified txt file instead of stdin
-o --output : Specify the output location for the generated result. Provide a name for the file (-o Result)
-n --no-output : Turn off report generation. This will only output to stdout
