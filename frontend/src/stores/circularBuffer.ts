export interface LogEntry {
  id: number
  html: string
  type: 'system' | 'stats'
}

export interface ILogBuffer {
  readonly length: number
  push(item: LogEntry): void
  getAt(index: number): LogEntry | undefined
  clear(): void
  onChange(fn: () => void): void
  offChange(fn: () => void): void
}

export class CircularBuffer implements ILogBuffer {
  private buffer: LogEntry[]
  private head = 0
  private _count = 0
  private listeners = new Set<() => void>()
  private capacity: number

  constructor(capacity: number) {
    this.capacity = capacity
    this.buffer = new Array(capacity)
  }

  push(item: LogEntry) {
    this.buffer[this.head] = item
    this.head = (this.head + 1) % this.capacity
    if (this._count < this.capacity) this._count++
    this.listeners.forEach(fn => fn())
  }

  get length(): number {
    return this._count
  }

  getAt(index: number): LogEntry | undefined {
    if (index < 0 || index >= this._count) return undefined
    const start = (this.head - this._count + this.capacity) % this.capacity
    return this.buffer[(start + index) % this.capacity]
  }

  clear() {
    this._count = 0
    this.head = 0
    this.listeners.forEach(fn => fn())
  }

  onChange(fn: () => void) {
    this.listeners.add(fn)
  }

  offChange(fn: () => void) {
    this.listeners.delete(fn)
  }
}
