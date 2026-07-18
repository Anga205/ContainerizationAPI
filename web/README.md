# Anga's Online Compiler

A browser-based compiler for **C, C++, Python, and Java**.  

## How to use

1. Go to the [website](https://compiler.anga.codes/)
2. Select a language.
3. Write or paste code in the editor.
4. Add stdin input (optional).
5. Run the program.
6. View stdout/stderr output from the compiler runtime.

## Execution Constraints

Code is executed in a controlled environment with fixed resource limits that can be changed via the UI.
- Limited CPU time (terminates after 60 seconds)
- Limited memory usage (you can use upto 128MB)

## How does this execute the code?

This is just the code for the frontend of the online compiler, the actual code execution is handled by the [ContainerizationAPI](https://github.com/Anga205/ContainerizationAPI) that i had made earlier, you can go to that repo to see that code, it essentially just uses unix tools to containerize your code and run it in a sandbox environment