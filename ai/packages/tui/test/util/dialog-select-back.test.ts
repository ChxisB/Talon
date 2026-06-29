import { describe, expect, it } from "bun:test"
import { shouldEnableDialogBackspace } from "../../src/ui/dialog-select-back"

describe("dialog select backspace behavior", () => {
  it("enables backspace when a back handler exists and filter input is not focused", () => {
    expect(shouldEnableDialogBackspace({ hasOnBack: true, filterInputFocused: false })).toBe(true)
  })

  it("disables backspace navigation while typing in filter input", () => {
    expect(shouldEnableDialogBackspace({ hasOnBack: true, filterInputFocused: true })).toBe(false)
  })

  it("disables backspace if there is no back handler", () => {
    expect(shouldEnableDialogBackspace({ hasOnBack: false, filterInputFocused: false })).toBe(false)
  })
})
