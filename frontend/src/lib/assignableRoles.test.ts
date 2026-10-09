import { describe, it, expect } from 'vitest'
import { assignableRoles, userErrorKey } from './assignableRoles'

const roles = [
  { id: '1', name: 'super_admin' },
  { id: '2', name: 'department_admin' },
  { id: '3', name: 'examiner' },
  { id: '4', name: 'employee' },
  { id: '5', name: 'qa_lead', permissions: ['questions:read', 'exams:read'] },
  { id: '6', name: 'big', permissions: ['questions:read', 'users:manage'] },
  { id: '7', name: 'sens', permissions: ['roles:read'] },
  { id: '8', name: 'nopermsknown' },
]
const names = (c: string | undefined, p?: string[]) => assignableRoles(c, p, roles).map((r) => r.name)

describe('assignableRoles', () => {
  it('super_admin may assign every role', () => {
    expect(names('super_admin')).toHaveLength(roles.length)
  })
  it('department_admin: only examiner and employee (no peer, no super_admin, no unknown-permission custom)', () => {
    expect(names('department_admin')).toEqual(['examiner', 'employee'])
  })
  it('department_admin with known permissions also gets covered custom roles, never sensitive ones', () => {
    expect(names('department_admin', ['questions:read', 'exams:read', 'roles:read'])).toEqual([
      'examiner', 'employee', 'qa_lead',
    ])
  })
  it('examiner: only employee', () => {
    expect(names('examiner')).toEqual(['employee'])
  })
  it('employee and unknown caller: nothing', () => {
    expect(names('employee')).toEqual([])
    expect(names(undefined)).toEqual([])
  })
  it('custom caller: no admin built-ins, only permission-subset roles', () => {
    expect(names('user_helper', ['questions:read', 'exams:read'])).toEqual(['qa_lead'])
    expect(names('user_helper')).toEqual([])
  })
})

describe('userErrorKey', () => {
  it('maps FORBIDDEN only', () => {
    expect(userErrorKey({ code: 'FORBIDDEN' })).toBe('users.messages.forbidden')
    expect(userErrorKey({ code: 'X' })).toBeNull()
    expect(userErrorKey(new Error('x'))).toBeNull()
  })
})
