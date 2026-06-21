import { extend } from "@tui/solid/components"
import { QRCodeRenderable } from "./renderables/QRCode.js"

declare module "@tui/solid" {
  interface OpenTUIComponents {
    qr_code: typeof QRCodeRenderable
  }
}

export function registerQRCode(): void {
  extend({ qr_code: QRCodeRenderable })
}

export * from "./index.js"
