import { render } from "@tui/solid"
import { ConsolePosition } from "@tui/core"
import ExampleSelector from "./components/ExampleSelector.js"

// Uncomment to debug solidjs reconciler
// process.env.DEBUG = "true"

const App = () => <ExampleSelector />

render(App, {
  targetFps: 30,
  exitOnCtrlC: false,
  consoleOptions: {
    position: ConsolePosition.BOTTOM,
    maxStoredLogs: 1000,
    sizePercent: 40,
  },
})
