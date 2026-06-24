import { createCliRenderer, TextAttributes } from "@tui/core"
import { createRoot } from "@tui/react"

const FONTS = ["tiny", "block", "slick", "pallet", "grid", "shade", "huge"] as const

export const App = () => {
  return (
    <box
      style={{
        padding: 1,
        flexDirection: "column",
        backgroundColor: "#0D1117",
      }}
    >
      <text
        content="TALON — ASCII Font Comparison"
        style={{ fg: "#58A6FF", attributes: TextAttributes.BOLD, marginBottom: 1 }}
      />

      {FONTS.map((font) => (
        <box
          key={font}
          style={{
            flexDirection: "column",
            marginBottom: 1,
          }}
        >
          <text
            content={`  font="${font}"`}
            style={{ fg: "#8B949E", attributes: TextAttributes.ITALIC, marginBottom: 0 }}
          />
          <ascii-font text="TALON" font={font} color="#00FF88" backgroundColor="transparent" selectable={false} />
        </box>
      ))}

      <text
        content="Use ↑/↓ to scroll, ctrl+c to quit"
        style={{ fg: "#484F58", marginTop: 1 }}
      />
    </box>
  )
}

if (import.meta.main) {
  const renderer = await createCliRenderer()
  createRoot(renderer).render(<App />)
}
