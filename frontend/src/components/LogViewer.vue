<script setup lang="ts">
import { ref, computed, onMounted, onUnmounted, nextTick } from 'vue'
import type { ILogBuffer, LogEntry } from '../stores/circularBuffer'

const props = defineProps<{
  buffer: ILogBuffer
  lineHeight: number
}>()

const viewportRef = ref<HTMLDivElement | null>(null)
const scrollTop = ref(0)
const viewportHeight = ref(600)
const overscan = 5
const version = ref(0)

const totalHeight = computed(() => {
  void version.value
  return props.buffer.length * props.lineHeight
})

const visibleRange = computed(() => {
  void version.value
  const start = Math.max(0, Math.floor(scrollTop.value / props.lineHeight) - overscan)
  const visibleCount = Math.ceil(viewportHeight.value / props.lineHeight)
  const end = Math.min(props.buffer.length, start + visibleCount + overscan * 2)
  return { start, end }
})

const visibleItems = computed(() => {
  void version.value
  const items: { index: number; entry: LogEntry }[] = []
  for (let i = visibleRange.value.start; i < visibleRange.value.end; i++) {
    const entry = props.buffer.getAt(i)
    if (entry) {
      items.push({ index: i, entry })
    }
  }
  return items
})

const offsetY = computed(() => visibleRange.value.start * props.lineHeight)

const isAtBottom = ref(true)

const onScroll = () => {
  const el = viewportRef.value
  if (!el) return
  scrollTop.value = el.scrollTop
  isAtBottom.value = el.scrollTop + el.clientHeight >= el.scrollHeight - 50
}

const scrollToBottom = () => {
  const el = viewportRef.value
  if (el) {
    el.scrollTop = el.scrollHeight
  }
}

const onDataChanged = () => {
  version.value++
  if (isAtBottom.value) {
    nextTick(() => scrollToBottom())
  }
}

onMounted(() => {
  nextTick(() => {
    const el = viewportRef.value
    if (el) {
      viewportHeight.value = el.clientHeight
    }
  })
  props.buffer.onChange(onDataChanged)
})

onUnmounted(() => {
  props.buffer.offChange(onDataChanged)
})
</script>

<template>
  <div class="log-card">
    <div class="log-header">
      <span class="log-title">日志输出</span>
    </div>
    <div ref="viewportRef" class="log-viewport" @scroll="onScroll">
      <div class="log-spacer" :style="{ height: totalHeight + 'px' }">
        <div class="log-visible" :style="{ transform: `translateY(${offsetY}px)` }">
          <div
            v-for="item in visibleItems"
            :key="item.index"
            class="log-line"
            v-html="item.entry.html"
          />
        </div>
      </div>
    </div>
  </div>
</template>

<style scoped>
.log-card {
  flex: 1;
  min-height: 0;
  background: #2B2930;
  border: 1px solid #454349;
  border-radius: 16px;
  overflow: hidden;
  display: flex;
  flex-direction: column;
}

.log-header {
  padding: 8px 16px;
  border-bottom: 1px solid #454349;
  flex-shrink: 0;
}

.log-title {
  font-size: 0.8125rem;
  font-weight: 500;
  color: #E6E1E5;
}

.log-viewport {
  flex: 1;
  min-height: 0;
  overflow-y: auto;
  overflow-x: hidden;
  background: #1C1B1F;
}

.log-viewport::-webkit-scrollbar {
  width: 8px;
}

.log-viewport::-webkit-scrollbar-track {
  background: transparent;
}

.log-viewport::-webkit-scrollbar-thumb {
  background: #454349;
  border-radius: 4px;
}

.log-viewport::-webkit-scrollbar-thumb:hover {
  background: #555359;
}

.log-spacer {
  position: relative;
  width: 100%;
}

.log-visible {
  position: absolute;
  top: 0;
  left: 0;
  right: 0;
}

.log-line {
  padding: 2px 8px;
  border-radius: 4px;
  font-family: "JetBrains Mono", "Fira Code", "Cascadia Code", monospace;
  font-size: 0.875rem;
  line-height: 20px;
  word-break: break-all;
  text-align: left;
  transition: background-color 100ms ease;
}

.log-line:hover {
  background: #36343B;
}
</style>

<style>
.log-line .log-idx {
  color: #79747E;
  user-select: none;
  margin-right: 8px;
  font-size: 0.75rem;
  opacity: 0.7;
}

.log-line .log-success { color: #A8DAB5; }
.log-line .log-failure { color: #F2B8B5; }
.log-line .log-info { color: #93C5FD; }
.log-line .log-warn { color: #FFD699; }
.log-line .log-default { color: #CAC4D0; }
.log-line .log-rate { color: #93C5FD; }
</style>
