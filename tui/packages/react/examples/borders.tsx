import { createCliRenderer } from "@tui/core"
import { createRoot } from "@tui/react"

export const App = () => {
  return (
    <>
      <box flexDirection="row">
        <box border borderStyle="single">
          <text content="Single" />
        </box>
        <box border borderStyle="double">
          <text content="Double" />
        </box>
        <box border borderStyle="rounded">
          <text content="Rounded" />
        </box>
        <box border borderStyle="heavy">
          <text content="Heavy" />
        </box>
      </box>
    </>
  )
}

if (import.meta.main) {
  const renderer = await createCliRenderer()
  createRoot(renderer).render(<App />)
}
