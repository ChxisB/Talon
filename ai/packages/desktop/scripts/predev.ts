import { $ } from "bun"

await $`bun ./scripts/copy-icons.ts ${process.env.TALON_CHANNEL ?? "dev"}`

await $`cd ../talon && bun script/build-node.ts`
