import { afterEach, describe, expect, test } from "bun:test"
import { Option, Redacted } from "effect"
import { Flag } from "@talon-ai/core/flag/flag"
import { ServerAuth } from "../../src/server/auth"

const original = {
  TALON_SERVER_PASSWORD: Flag.TALON_SERVER_PASSWORD,
  TALON_SERVER_USERNAME: Flag.TALON_SERVER_USERNAME,
}

afterEach(() => {
  Flag.TALON_SERVER_PASSWORD = original.TALON_SERVER_PASSWORD
  Flag.TALON_SERVER_USERNAME = original.TALON_SERVER_USERNAME
})

describe("ServerAuth", () => {
  test("does not emit auth headers without a password", () => {
    Flag.TALON_SERVER_PASSWORD = undefined
    Flag.TALON_SERVER_USERNAME = "alice"

    expect(ServerAuth.header()).toBeUndefined()
    expect(ServerAuth.headers()).toBeUndefined()
  })

  test("defaults to the talon username", () => {
    Flag.TALON_SERVER_PASSWORD = "secret"
    Flag.TALON_SERVER_USERNAME = undefined

    expect(ServerAuth.headers()).toEqual({
      Authorization: `Basic ${Buffer.from("talon:secret").toString("base64")}`,
    })
  })

  test("uses the configured username", () => {
    Flag.TALON_SERVER_PASSWORD = "secret"
    Flag.TALON_SERVER_USERNAME = "alice"

    expect(ServerAuth.headers()).toEqual({
      Authorization: `Basic ${Buffer.from("alice:secret").toString("base64")}`,
    })
  })

  test("prefers explicit credentials", () => {
    Flag.TALON_SERVER_PASSWORD = "secret"
    Flag.TALON_SERVER_USERNAME = "alice"

    expect(ServerAuth.headers({ password: "cli-secret", username: "bob" })).toEqual({
      Authorization: `Basic ${Buffer.from("bob:cli-secret").toString("base64")}`,
    })
  })

  test("validates decoded credentials against effect config", () => {
    const config = { password: Option.some("secret"), username: "alice" }

    expect(ServerAuth.required(config)).toBe(true)
    expect(ServerAuth.authorized({ username: "alice", password: Redacted.make("secret") }, config)).toBe(true)
    expect(ServerAuth.authorized({ username: "talon", password: Redacted.make("secret") }, config)).toBe(false)
  })
})
