/**
 * Setup project of the visual suite (FR-BB320 AC-13): creates the page fixtures once per run, before any page test.
 *
 * It is a project, not a beforeAll: a worker restarts after a failed test and reruns beforeAll, which created the exams a
 * second time (duplicate cards on the portal). The ids go to .auth/visual-fixtures.json, and the database is new each run.
 */
import { test as setup } from '@playwright/test'
import fs from 'fs'
import { createTestExam, createTestQuestion, getSeedData } from './fixtures/seed'
import { VISUAL_FIXTURES_FILE, passedSession, startSession, type VisualFixtures } from './fixtures/visual'

setup('create the visual-page fixtures', async () => {
  const seed = await getSeedData()
  const question = await createTestQuestion(seed.adminToken, 'Visual question', 'single', true)
  const taking = await createTestExam(seed.adminToken, 'Visual exam (take)', question.id, {
    assignToUserId: seed.employeeId,
  })
  const result = await createTestExam(seed.adminToken, 'Visual exam (result)', question.id, {
    assignToUserId: seed.employeeId,
  })
  const fixtures: VisualFixtures = {
    takingSessionId: await startSession(taking.id),
    resultSessionId: await passedSession(result.id),
    questionId: question.id,
    examIds: [taking.id, result.id],
  }
  fs.writeFileSync(VISUAL_FIXTURES_FILE, JSON.stringify(fixtures), 'utf8')
})
