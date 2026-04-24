import { defineConfig, devices } from "@playwright/test"
import { defineBddConfig } from "playwright-bdd"

function e2eSlowMo() {
  const value = Number(process.env.E2E_SLOW_MO ?? "220")
  return Number.isFinite(value) && value >= 0 ? value : 220
}

function e2eWorkers() {
  const value = Number(process.env.E2E_WORKERS ?? "1")
  return Number.isInteger(value) && value > 0 ? value : 1
}

const testDir = defineBddConfig({
  features: "e2e/features/**/*.feature",
  steps: ["e2e/fixtures/**/*.ts", "e2e/steps/**/*.ts"],
  outputDir: "e2e/.generated/agentic-itsm",
  language: "zh-CN",
  missingSteps: "fail-on-gen",
})

export default defineConfig({
  testDir,
  timeout: 180_000,
  workers: e2eWorkers(),
  expect: {
    timeout: 5_000,
  },
  use: {
    baseURL: process.env.E2E_BASE_URL ?? "http://localhost:3000",
    headless: false,
    launchOptions: {
      slowMo: e2eSlowMo(),
    },
    trace: "on-first-retry",
  },
  projects: [
    {
      name: "chromium",
      use: { ...devices["Desktop Chrome"], channel: "chrome" },
    },
  ],
})
