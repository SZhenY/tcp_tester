import { defineStore } from 'pinia'
import { ref, computed } from 'vue'
import { EventsOn } from '../../wailsjs/runtime/runtime'
import { StartTest, StopTest, ResolveDomain, GetDefaultConfig } from '../../wailsjs/go/main/App'
import { CircularBuffer } from './circularBuffer'

// HTTP GET 请求工具，带超时
async function httpGet(url: string): Promise<string> {
  const res = await fetch(url, { signal: AbortSignal.timeout(5000) })
  return (await res.text()).trim()
}

// IP 遮罩函数
function maskIP(ip: string): string {
  if (ip.includes(':')) {
    const p = ip.split(':')
    if (p.length > 4) {
      return p.slice(0, 2).join(':') + ':****:****:' + p.slice(-2).join(':')
    }
    return '****:' + p.slice(-2).join(':')
  }
  const p = ip.split('.')
  if (p.length === 4) {
    return p[0] + '.***.***.' + p[3]
  }
  return '***.***.***'
}

// API 轮次定义
const IP_API_ROUNDS = [
  { ipv4: 'https://api-ipv4.ip.sb/ip', ipv6: 'https://api-ipv6.ip.sb/ip' },
  { ipv4: 'https://ipinfo.io/ip', ipv6: 'https://api64.ipify.org' },
]

export const useTesterStore = defineStore('tester', () => {
  const domain = ref('')
  const port = ref('80')
  const ipOptions = ref<{ label: string; value: string; type: 'ipv4' | 'ipv6' }[]>([])
  const selectedIp = ref('')
  const threadCount = ref('64')
  const intervalMs = ref('0')
  const failureLimit = ref('50')
  const successLimit = ref('1000')

  const isRunning = ref(false)
  const logBuffer = new CircularBuffer(2000)
  let logIdCounter = 0

  const stats = ref({ success: 0, failure: 0, total: 0, rate: 0, avgCps: 0 })
  const displayStats = ref({ success: 0, failure: 0, total: 0, rate: 0, avgCps: 0 })
  const prevStats = ref({ success: 0, failure: 0 })

  let statsTimer: ReturnType<typeof setInterval> | null = null

  // 本机公网 IP 检测
  const publicIPv4 = ref('')
  const publicIPv6 = ref('')
  const hideIP = ref(true)

  // 从后端同步默认配置
  const initDefaults = async () => {
    try {
      const cfg = await GetDefaultConfig()
      port.value = cfg.port
      threadCount.value = cfg.threadCount
      intervalMs.value = cfg.intervalMs
      failureLimit.value = cfg.failureLimit
      successLimit.value = cfg.successLimit
    } catch {
      // 回退：使用硬编码默认值
    }
  }

  const isIdle = computed(() => !isRunning.value)

  // IP 状态
  const ipStatus = computed(() => {
    const has4 = !!publicIPv4.value
    const has6 = !!publicIPv6.value
    if (has4 && has6) return { text: '就绪', color: '#A8DAB5' }
    if (has4 && !has6) return { text: 'IPv6 缺失', color: '#FFD699' }
    if (!has4 && has6) return { text: 'IPv4 缺失', color: '#FFD699' }
    return { text: '无 IP，请检查网络', color: '#F2B8B5' }
  })

  // 遮罩后的 IP 显示
  const maskedIPv4 = computed(() => {
    if (!publicIPv4.value) return ''
    return hideIP.value ? maskIP(publicIPv4.value) : publicIPv4.value
  })
  const maskedIPv6 = computed(() => {
    if (!publicIPv6.value) return ''
    return hideIP.value ? maskIP(publicIPv6.value) : publicIPv6.value
  })

  // 查询本机公网 IP（循环重试，5 秒间隔，ip.sb ↔ ipinfo/ipify 交替，最多重试 10 次）
  const fetchPublicIP = async () => {
    const maxRetries = 10
    for (let round = 0; round < maxRetries; round++) {
      const apis = IP_API_ROUNDS[round % IP_API_ROUNDS.length]
      const [ipv4Res, ipv6Res] = await Promise.allSettled([
        httpGet(apis.ipv4),
        httpGet(apis.ipv6),
      ])

      const ipv4 = ipv4Res.status === 'fulfilled' ? ipv4Res.value : ''
      const ipv6 = ipv6Res.status === 'fulfilled' ? ipv6Res.value : ''

      if (ipv4) publicIPv4.value = ipv4
      if (ipv6) publicIPv6.value = ipv6

      // 两个都获取到了，跳出
      if (publicIPv4.value && publicIPv6.value) break

      // 至少一个还没获取到，等待 5 秒后重试
      if (round < maxRetries - 1) {
        await new Promise(resolve => setTimeout(resolve, 5000))
      }
    }
  }

  const pushSystemLog = (html: string) => {
    logBuffer.push({ id: ++logIdCounter, html, type: 'system' })
  }

  const pushStatsLog = (html: string) => {
    logBuffer.push({ id: ++logIdCounter, html, type: 'stats' })
  }

  const formatTimestamp = () => {
    const now = new Date()
    const h = String(now.getHours()).padStart(2, '0')
    const m = String(now.getMinutes()).padStart(2, '0')
    const s = String(now.getSeconds()).padStart(2, '0')
    return `${h}:${m}:${s}`
  }

  const getLogClass = (text: string): string => {
    if (text.includes('每秒新增连接数')) return 'log-info'
    if (text.includes('成功') || text.includes('恢复') || text.includes('已断开')) return 'log-success'
    if (text.includes('失败') || text.includes('错误') || text.includes('原因')) return 'log-failure'
    if (text.includes('解析') || text.includes('开始') || text.includes('连接池')) return 'log-info'
    if (text.includes('停止') || text.includes('正在断开')) return 'log-warn'
    return 'log-default'
  }

  const startStatsTimer = () => {
    stopStatsTimer()
    statsTimer = setInterval(() => {
      if (!isRunning.value) return
      const rate = stats.value.rate
      if (rate > 0) {
        const halfRate = Math.round(rate * 0.5)
        displayStats.value = {
          ...stats.value,
          success: stats.value.success + halfRate,
          total: stats.value.total + halfRate,
        }
      } else {
        displayStats.value = { ...stats.value }
      }
    }, 500)
  }

  const stopStatsTimer = () => {
    if (statsTimer) {
      clearInterval(statsTimer)
      statsTimer = null
    }
  }

  const resolveDomainAction = async () => {
    if (!domain.value.trim()) return
    try {
      const res = await ResolveDomain(domain.value.trim())
      const lines = res.split('\n').filter(Boolean)
      const options: { label: string; value: string; type: 'ipv4' | 'ipv6' }[] = []
      let currentType: 'ipv4' | 'ipv6' = 'ipv4'

      for (const line of lines) {
        const trimmed = line.trim()
        if (trimmed === 'IPv4:') {
          currentType = 'ipv4'
        } else if (trimmed === 'IPv6:') {
          currentType = 'ipv6'
        } else {
          options.push({
            label: `[${currentType.toUpperCase()}] ${trimmed}`,
            value: trimmed,
            type: currentType
          })
        }
      }

      ipOptions.value = options
      const firstIpv4 = options.find(o => o.type === 'ipv4')
      selectedIp.value = firstIpv4 ? firstIpv4.value : (options[0]?.value || '')

      pushSystemLog(`<span class="log-info">[${formatTimestamp()}] 域名解析完成: ${domain.value} -> ${options.length} 个地址</span>`)
    } catch (err: any) {
      pushSystemLog(`<span class="log-failure">[${formatTimestamp()}] 解析失败: ${err}</span>`)
    }
  }

  const startTestAction = async () => {
    const trimmedDomain = domain.value.trim()
    if (!trimmedDomain) {
      pushSystemLog(`<span class="log-failure">[${formatTimestamp()}] 错误: 请先输入域名</span>`)
      return
    }

    // 如果未解析过，自动解析
    let ip = selectedIp.value
    if (!ip || ipOptions.value.length === 0) {
      await resolveDomainAction()
      ip = selectedIp.value
    }

    if (!ip) {
      pushSystemLog(`<span class="log-failure">[${formatTimestamp()}] 错误: 域名解析失败，无法获取 IP</span>`)
      return
    }

    const target = ip.includes(':') ? `[${ip}]:${port.value}` : `${ip}:${port.value}`
    try {
      await StartTest(target, Number(threadCount.value), Number(intervalMs.value), Number(failureLimit.value), Number(successLimit.value))
      isRunning.value = true
      logBuffer.clear()
      logIdCounter = 0
      stats.value = { success: 0, failure: 0, total: 0, rate: 0, avgCps: 0 }
      displayStats.value = { success: 0, failure: 0, total: 0, rate: 0, avgCps: 0 }
      prevStats.value = { success: 0, failure: 0 }
      startStatsTimer()
    } catch (err: any) {
      pushSystemLog(`<span class="log-failure">[${formatTimestamp()}] 启动失败: ${err}</span>`)
    }
  }

  const stopTestAction = async () => {
    await StopTest()
    isRunning.value = false
    stopStatsTimer()
    displayStats.value = { ...stats.value }
  }

  const setupListeners = () => {
    EventsOn('log', (msg: string) => {
      const cls = getLogClass(msg)
      pushSystemLog(`<span class="${cls}">[${formatTimestamp()}] ${msg.replace(/^\[\d{2}:\d{2}:\d{2}(\.\d{3})?\]\s*/, '')}</span>`)
    })

    EventsOn('stats', (data: any) => {
      if (isRunning.value) {
        const newSucc = data.success - prevStats.value.success
        const newFail = data.failure - prevStats.value.failure
        if (newSucc > 0 || newFail > 0) {
          pushStatsLog(
            `<span class="log-default">[${formatTimestamp()}]</span> ` +
            `<span class="log-success">总成功: ${data.success} (+${newSucc})</span> ` +
            `<span class="log-failure">总失败: ${data.failure} (+${newFail})</span> ` +
            `<span class="log-rate">每秒新增连接数(CPS): ${Math.round(data.rate)}</span>`
          )
        }
        prevStats.value = { success: data.success, failure: data.failure }
      }
      stats.value = {
        success: data.success,
        failure: data.failure,
        total: data.total,
        rate: data.rate,
        avgCps: data.avgCps || 0
      }
      displayStats.value = { ...stats.value }
    })

    EventsOn('testFinished', () => {
      isRunning.value = false
      stopStatsTimer()
      displayStats.value = { ...stats.value }
    })
  }

  return {
    domain, port, ipOptions, selectedIp, threadCount, intervalMs, failureLimit, successLimit,
    isRunning, logBuffer, stats, displayStats, isIdle,
    publicIPv4, publicIPv6, hideIP, ipStatus, maskedIPv4, maskedIPv6, fetchPublicIP, initDefaults,
    resolveDomainAction, startTestAction, stopTestAction, setupListeners
  }
})
