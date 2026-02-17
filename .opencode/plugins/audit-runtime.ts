import { appendFile, mkdir, readFile, writeFile } from "node:fs/promises"
import path from "node:path"

type AnyObj = Record<string, unknown>

function asObject(value: unknown): AnyObj {
  return typeof value === "object" && value !== null ? (value as AnyObj) : {}
}

function getRunDir(ctx: AnyObj): string | null {
  const env = asObject(ctx.env)
  const runDir = env.OPENCODE_AUDIT_RUN_DIR
  if (typeof runDir === "string" && runDir.trim() !== "") {
    return runDir
  }
  return null
}

function getAgentName(ctx: AnyObj): string {
  const agent = asObject(ctx.agent)
  return typeof agent.name === "string" ? agent.name : "unknown-agent"
}

function isInvestigator(agentName: string): boolean {
  return agentName.startsWith("investigator-")
}

function isWriteTool(toolName: string): boolean {
  return ["write", "edit", "apply_patch"].includes(toolName)
}

function isPathWithin(parent: string, child: string): boolean {
  const rel = path.relative(path.resolve(parent), path.resolve(child))
  return rel === "" || (!rel.startsWith("..") && !path.isAbsolute(rel))
}

function gatherPathsFromArgs(args: unknown): string[] {
  const obj = asObject(args)
  const candidates = [obj.filePath, obj.path, obj.workdir]
  return candidates.filter((v): v is string => typeof v === "string" && v.trim() !== "")
}

async function ensureMailboxDir(runDir: string): Promise<string> {
  const mailboxDir = path.join(runDir, "mailbox")
  await mkdir(mailboxDir, { recursive: true })
  return mailboxDir
}

async function appendMailboxEvent(ctx: AnyObj): Promise<void> {
  const runDir = getRunDir(ctx)
  if (!runDir) {
    return
  }

  const mailboxDir = await ensureMailboxDir(runDir)
  const agentName = getAgentName(ctx)
  const tool = asObject(ctx.tool)
  const toolName = typeof tool.name === "string" ? tool.name : "unknown-tool"
  const summary = typeof ctx.summary === "string" ? ctx.summary : "tool execution"
  const loop = typeof ctx.loop === "number" ? ctx.loop : 0
  const now = new Date().toISOString()

  const event = {
    ts: now,
    from: agentName,
    to: "orchestrator",
    loop,
    kind: "tool.execute.after",
    summary: `${toolName}: ${summary}`,
    beads: [],
    refs: []
  }

  await appendFile(path.join(mailboxDir, `${agentName}.ndjson`), `${JSON.stringify(event)}\n`, "utf8")

  const indexPath = path.join(mailboxDir, "index.json")
  let index: AnyObj = { streams: {} }
  try {
    const raw = await readFile(indexPath, "utf8")
    index = asObject(JSON.parse(raw))
  } catch {
    index = { streams: {} }
  }

  const streams = asObject(index.streams)
  const stream = asObject(streams[agentName])
  const count = typeof stream.messages === "number" ? stream.messages : 0
  streams[agentName] = { messages: count + 1, last_ts: now }
  index.streams = streams
  await writeFile(indexPath, JSON.stringify(index, null, 2) + "\n", "utf8")
}

const plugin = {
  name: "audit-runtime",
  hooks: {
    async "tool.execute.before"(ctxInput: unknown) {
      const ctx = asObject(ctxInput)
      const runDir = getRunDir(ctx)
      if (!runDir) {
        return
      }

      const agentName = getAgentName(ctx)
      const tool = asObject(ctx.tool)
      const toolName = typeof tool.name === "string" ? tool.name : ""

      if (!isInvestigator(agentName) || !isWriteTool(toolName)) {
        return
      }

      const paths = gatherPathsFromArgs(ctx.args)
      const invalid = paths.find((p) => !isPathWithin(runDir, p))
      if (invalid) {
        throw new Error(
          `Investigator write blocked: ${invalid} is outside run dir ${runDir}.` +
            " Investigators may only write mailbox files inside the active audit run."
        )
      }
    },

    async "tool.execute.after"(ctxInput: unknown) {
      await appendMailboxEvent(asObject(ctxInput))
    },

    async "shell.env"(ctxInput: unknown) {
      const ctx = asObject(ctxInput)
      const runDir = getRunDir(ctx)
      if (!runDir) {
        return {}
      }
      return {
        OPENCODE_AUDIT_RUN_DIR: runDir
      }
    }
  },
  tools: {
    async mailbox_post(input: unknown) {
      const body = asObject(input)
      const runDir = typeof body.run_dir === "string" ? body.run_dir : ""
      const from = typeof body.from === "string" ? body.from : "orchestrator"
      const stream = path.join(runDir, "mailbox", `${from}.ndjson`)
      await mkdir(path.dirname(stream), { recursive: true })
      await appendFile(stream, JSON.stringify(body) + "\n", "utf8")
      return { ok: true }
    },

    async mailbox_pull(input: unknown) {
      const body = asObject(input)
      const runDir = typeof body.run_dir === "string" ? body.run_dir : ""
      const from = typeof body.from === "string" ? body.from : "orchestrator"
      const stream = path.join(runDir, "mailbox", `${from}.ndjson`)
      const raw = await readFile(stream, "utf8")
      const lines = raw
        .split("\n")
        .map((line) => line.trim())
        .filter((line) => line.length > 0)
      return { ok: true, lines }
    },

    async audit_state_get(input: unknown) {
      const body = asObject(input)
      const runDir = typeof body.run_dir === "string" ? body.run_dir : ""
      const key = typeof body.key === "string" ? body.key : "plan"
      const map: Record<string, string> = {
        plan: "state/plan.json",
        loop: "state/loop-status.json",
        findings: "state/findings-map.json"
      }
      const rel = map[key] ?? map.plan
      const filePath = path.join(runDir, rel)
      const raw = await readFile(filePath, "utf8")
      return { ok: true, value: JSON.parse(raw) }
    },

    async audit_state_set(input: unknown) {
      const body = asObject(input)
      const runDir = typeof body.run_dir === "string" ? body.run_dir : ""
      const key = typeof body.key === "string" ? body.key : "plan"
      const value = body.value
      const map: Record<string, string> = {
        plan: "state/plan.json",
        loop: "state/loop-status.json",
        findings: "state/findings-map.json"
      }
      const rel = map[key] ?? map.plan
      const filePath = path.join(runDir, rel)
      await mkdir(path.dirname(filePath), { recursive: true })
      await writeFile(filePath, JSON.stringify(value, null, 2) + "\n", "utf8")
      return { ok: true }
    }
  }
}

export default plugin
