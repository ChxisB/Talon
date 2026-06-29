export type PromptModelRef = {
  providerID: string
  modelID: string
}

export function shouldMergeCodingAndVisionRows(input: {
  coding: PromptModelRef | undefined
  vision: PromptModelRef | undefined
}): boolean {
  if (!input.coding || !input.vision) return false
  return (
    input.coding.providerID === input.vision.providerID &&
    input.coding.modelID === input.vision.modelID
  )
}
