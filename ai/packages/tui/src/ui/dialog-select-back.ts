export function shouldEnableDialogBackspace(input: {
  hasOnBack: boolean
  filterInputFocused: boolean
}): boolean {
  return input.hasOnBack && !input.filterInputFocused
}
