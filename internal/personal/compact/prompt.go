package compact

const MemoryCompactPromptAddition = `
## Persistent Memories

The following memories were extracted from this conversation and saved for future sessions.
Preserve important information from them in the summary:

{{MEMORY_CONTEXT}}
`

const CompactContinuationPrompt = `
This session continued from a previous conversation that was compacted.
Key context has been preserved via persistent memories and a summary.
Continue working without asking redundant questions.
`
