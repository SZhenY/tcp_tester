<script setup lang="ts">
import { ref, computed, onMounted } from 'vue'
import {
  NCard, NInput, NButton, NSelect, NSpace, NGrid, NGridItem,
  useMessage
} from 'naive-ui'
import { WindowMinimise, WindowToggleMaximise, Quit, WindowGetPosition, WindowSetPosition } from '../wailsjs/runtime/runtime'
import { useTesterStore } from './stores/tester'
import LogViewer from './components/LogViewer.vue'
import appIcon from './assets/images/app-icon.svg'

const store = useTesterStore()
const message = useMessage()
const selectOpen = ref(false)
const onSelectUpdate = (show: boolean) => { selectOpen.value = show }

onMounted(() => {
  store.setupListeners()
  store.initDefaults()
  store.fetchPublicIP()
})

const minimizeWindow = () => WindowMinimise()
const toggleMaximize = () => WindowToggleMaximise()
const closeWindow = () => Quit()

// 窗口拖拽
const isDragging = ref(false)
const dragStart = ref({ x: 0, y: 0 })
const winStart = ref({ x: 0, y: 0 })

const startDrag = async (e: MouseEvent) => {
  const target = e.target as HTMLElement
  if (target.closest('.window-controls') || target.closest('.header-ip-row')) return

  isDragging.value = true
  dragStart.value = { x: e.screenX, y: e.screenY }
  const pos = await WindowGetPosition()
  winStart.value = { x: pos.x, y: pos.y }

  document.addEventListener('mousemove', onDrag)
  document.addEventListener('mouseup', stopDrag)
}

const onDrag = (e: MouseEvent) => {
  if (!isDragging.value) return
  const dx = e.screenX - dragStart.value.x
  const dy = e.screenY - dragStart.value.y
  WindowSetPosition(winStart.value.x + dx, winStart.value.y + dy)
}

const stopDrag = () => {
  isDragging.value = false
  document.removeEventListener('mousemove', onDrag)
  document.removeEventListener('mouseup', stopDrag)
}

const formattedStats = computed(() => ({
  success: store.displayStats.success.toLocaleString(),
  failure: store.displayStats.failure.toLocaleString(),
  cps: Math.round(store.displayStats.avgCps).toLocaleString()
}))

const targetDisplay = computed(() => {
  return store.selectedIp || store.domain || '未设置'
})
</script>

<template>
  <div class="app-container">

    <div class="header-area card-appear" @mousedown="startDrag">
      <div class="header-main">
        <div class="header-title-row">
          <img :src="appIcon" alt="" class="app-icon" />
          <h1 class="app-title">TCP 并发连接测试工具</h1>
          <span class="status-tag" :style="{ color: store.ipStatus.color, borderColor: store.ipStatus.color }">
            {{ store.ipStatus.text }}
          </span>
        </div>
        <div class="header-ip-row">
          <span class="ip-item ip-item-ipv4">
            <span class="ip-label">🔒 IPv4</span>
            <span class="ip-value" :class="{ 'ip-empty': !store.maskedIPv4 }">
              {{ store.maskedIPv4 || '未检测到' }}
            </span>
          </span>
          <span class="ip-item ip-item-ipv6">
            <span class="ip-label">🔒 IPv6</span>
            <span class="ip-value" :class="{ 'ip-empty': !store.maskedIPv6 }">
              {{ store.maskedIPv6 || '未检测到' }}
            </span>
          </span>
          <label class="ip-checkbox">
            <input type="checkbox" v-model="store.hideIP" />
            <span>隐藏</span>
          </label>
        </div>
      </div>
      <div class="window-controls">
        <button class="win-btn" @click="minimizeWindow" title="最小化">
          <svg width="12" height="12" viewBox="0 0 12 12"><rect y="5" width="12" height="2" fill="currentColor"/></svg>
        </button>
        <button class="win-btn" @click="toggleMaximize" title="最大化">
          <svg width="12" height="12" viewBox="0 0 12 12"><rect x="1" y="1" width="10" height="10" stroke="currentColor" stroke-width="1.5" fill="none"/></svg>
        </button>
        <button class="win-btn win-btn-close" @click="closeWindow" title="关闭">
          <svg width="12" height="12" viewBox="0 0 12 12"><path d="M1 1L11 11M11 1L1 11" stroke="currentColor" stroke-width="1.5"/></svg>
        </button>
      </div>
    </div>

    <n-card
      title="目标设置"
      class="card-base card-appear card-appear-delay-1"
      :bordered="true"
    >
      <n-grid :cols="24" :x-gap="10" :y-gap="10">
        <n-grid-item :span="7">
          <div class="field-label">域名</div>
          <n-input
            v-model:value="store.domain"
            placeholder="例如 www.example.com"
            :disabled="store.isRunning"
            size="small"
          />
        </n-grid-item>
        <n-grid-item :span="3">
          <div class="field-label">端口</div>
          <n-input
            v-model:value="store.port"
            placeholder="80"
            :disabled="store.isRunning"
            size="small"
          />
        </n-grid-item>
        <n-grid-item :span="10">
          <div class="field-label">目标 IP</div>
          <n-select
            v-model:value="store.selectedIp"
            :options="store.ipOptions"
            placeholder="解析后选择或直接输入 IP"
            :show="selectOpen"
            @update:show="onSelectUpdate"
            :disabled="store.isRunning"
            size="small"
          />
        </n-grid-item>
        <n-grid-item :span="4">
          <div class="field-label">&nbsp;</div>
          <n-button
            type="primary"
            @click="store.resolveDomainAction"
            :disabled="store.isRunning || !store.domain.trim()"
            block
            size="small"
          >
            解析
          </n-button>
        </n-grid-item>

        <n-grid-item :span="6">
          <div class="field-label">并发线程数</div>
          <n-input
            v-model:value="store.threadCount"
            placeholder="100"
            :disabled="store.isRunning"
            size="small"
          />
        </n-grid-item>
        <n-grid-item :span="6">
          <div class="field-label">发送间隔 (ms)</div>
          <n-input
            v-model:value="store.intervalMs"
            placeholder="1"
            :disabled="store.isRunning"
            size="small"
          />
        </n-grid-item>
        <n-grid-item :span="6">
          <div class="field-label">失败上限 (停止)</div>
          <n-input
            v-model:value="store.failureLimit"
            placeholder="50"
            :disabled="store.isRunning"
            size="small"
          />
        </n-grid-item>
        <n-grid-item :span="6">
          <div class="field-label">成功上限 (停止)</div>
          <n-input
            v-model:value="store.successLimit"
            placeholder="1000"
            :disabled="store.isRunning"
            size="small"
          />
        </n-grid-item>
      </n-grid>
    </n-card>

    <n-card class="card-base card-appear card-appear-delay-2">
      <div class="actions-row">
        <n-space :size="8">
          <n-button
            type="primary"
            :disabled="!store.isIdle || !store.domain.trim()"
            @click="store.startTestAction"
            size="small"
          >
            开始
          </n-button>
          <n-button
            type="error"
            :disabled="store.isIdle"
            @click="store.stopTestAction"
            size="small"
          >
            停止
          </n-button>
          <span class="target-info">
            <span class="target-tag">
              <span class="tag-label">目标</span>
              <span class="tag-value">{{ targetDisplay }}:{{ store.port }}</span>
            </span>
            <span class="target-tag" v-if="store.threadCount">
              <span class="tag-label">并发</span>
              <span class="tag-value">{{ store.threadCount }}</span>
            </span>
          </span>
        </n-space>

        <n-space :size="16" class="stats-area">
          <div class="stat-item">
            <span class="stat-label">成功</span>
            <span class="stat-value stat-success">{{ formattedStats.success }}</span>
          </div>
          <div class="stat-item">
            <span class="stat-label">失败</span>
            <span class="stat-value stat-failure">{{ formattedStats.failure }}</span>
          </div>
          <div class="stat-item">
            <span class="stat-label">CPS</span>
            <span class="stat-value stat-rate">{{ formattedStats.cps }}</span>
          </div>
        </n-space>
      </div>
    </n-card>

    <LogViewer :buffer="store.logBuffer" :line-height="20" />

  </div>
</template>

<style scoped>
.app-container {
  height: 100vh;
  width: 100vw;
  background: #1C1B1F;
  color: #E6E1E5;
  padding: 0 16px 14px 16px;
  display: flex;
  flex-direction: column;
  gap: 10px;
  overflow: hidden;
  font-family: "Nunito", -apple-system, BlinkMacSystemFont, "Segoe UI", sans-serif;
}

.header-area {
  display: flex;
  justify-content: space-between;
  align-items: flex-start;
  padding: 8px 0 4px 0;
  flex-shrink: 0;
  cursor: default;
  user-select: none;
}

.header-main {
  display: flex;
  flex-direction: column;
  gap: 3px;
}

.header-title-row {
  display: flex;
  align-items: center;
  gap: 8px;
}

.app-icon {
  width: 24px;
  height: 24px;
  border-radius: 6px;
  flex-shrink: 0;
}

.app-title {
  font-size: 1rem;
  font-weight: 600;
  color: #D0BCFF;
  letter-spacing: 0;
  margin: 0;
}

.status-tag {
  font-size: 0.6875rem;
  font-weight: 500;
  padding: 1px 8px;
  border-radius: 6px;
  border: 1px solid;
  line-height: 1.4;
}

.header-ip-row {
  display: flex;
  align-items: center;
  gap: 8px;
  margin-top: 4px;
  padding: 5px 10px;
  background: rgba(255, 255, 255, 0.03);
  border-radius: 10px;
  border: 1px solid rgba(255, 255, 255, 0.06);
  min-height: 28px;
}

.ip-item {
  display: flex;
  align-items: center;
  gap: 6px;
  padding: 2px 10px;
  background: rgba(147, 197, 253, 0.06);
  border-radius: 8px;
  overflow: hidden;
}

.ip-item-ipv4 {
  min-width: 20ch;
  max-width: 25ch;
}

.ip-item-ipv6 {
  min-width: 39ch;
  max-width: 45ch;
}

.ip-label {
  font-weight: 600;
  font-size: 0.625rem;
  text-transform: uppercase;
  letter-spacing: 0.08em;
  color: #93C5FD;
  opacity: 0.7;
  flex-shrink: 0;
}

.ip-value {
  font-family: "Cascadia Code", "JetBrains Mono", "Fira Code", monospace;
  color: #E6E1E5;
  font-size: 0.75rem;
  font-weight: 400;
  letter-spacing: 0.03em;
  font-variant-numeric: tabular-nums;
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
  user-select: text;
}

.ip-empty {
  color: #79747E;
  font-style: italic;
}

.ip-checkbox {
  margin-left: auto;
  font-size: 0.625rem;
  color: #79747E;
  cursor: pointer;
  display: flex;
  align-items: center;
  gap: 4px;
  user-select: none;
  transition: color 100ms ease;
}

.ip-checkbox:hover {
  color: #CAC4D0;
}

.ip-checkbox input[type="checkbox"] {
  accent-color: #D0BCFF;
  width: 12px;
  height: 12px;
  cursor: pointer;
}

.window-controls {
  display: flex;
  gap: 0;
}

.win-btn {
  width: 36px;
  height: 28px;
  border: none;
  background: transparent;
  color: #CAC4D0;
  cursor: pointer;
  display: flex;
  align-items: center;
  justify-content: center;
  transition: background-color 100ms ease;
  padding: 0;
}

.win-btn:hover {
  background: rgba(255, 255, 255, 0.08);
}

.win-btn:active {
  background: rgba(255, 255, 255, 0.05);
}

.win-btn-close:hover {
  background: #C42B1C;
  color: white;
}

.win-btn-close:active {
  background: #A31D12;
  color: white;
}

.field-label {
  font-size: 0.6875rem;
  font-weight: 500;
  color: #CAC4D0;
  margin-bottom: 3px;
  letter-spacing: 0.01em;
}

.card-base {
  background: #2B2930 !important;
  border: 1px solid #454349 !important;
  border-radius: 16px !important;
  flex-shrink: 0;
}

.card-base :deep(.n-card__content) {
  padding: 12px 16px !important;
}

.card-base :deep(.n-card-header) {
  padding: 10px 16px 6px !important;
}

.card-base :deep(.n-card-header__main) {
  font-size: 0.8125rem !important;
  font-weight: 500 !important;
}

.actions-row {
  display: flex;
  justify-content: space-between;
  align-items: center;
}

.target-info {
  display: inline-flex;
  gap: 8px;
  margin-left: 8px;
}

.target-tag {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  padding: 3px 10px;
  background: rgba(208, 188, 255, 0.08);
  border: 1px solid rgba(208, 188, 255, 0.15);
  border-radius: 8px;
  font-size: 0.75rem;
}

.tag-label {
  color: #93C5FD;
  font-weight: 500;
  font-size: 0.6875rem;
}

.tag-value {
  color: #E6E1E5;
  font-weight: 600;
  font-family: "Cascadia Code", "JetBrains Mono", monospace;
}

.stats-area {
  display: flex;
  align-items: center;
}

.stat-item {
  display: flex;
  flex-direction: column;
  align-items: center;
  min-width: 50px;
}

.stat-label {
  font-size: 0.6875rem;
  color: #CAC4D0;
  margin-bottom: 1px;
}

.stat-value {
  font-size: 1.125rem;
  font-weight: 600;
  font-variant-numeric: tabular-nums;
  line-height: 1.2;
}

.stat-success { color: #A8DAB5; }
.stat-failure { color: #F2B8B5; }
.stat-rate { color: #93C5FD; }
</style>
