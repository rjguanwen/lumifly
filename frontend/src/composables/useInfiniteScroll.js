import { nextTick, onBeforeUnmount, ref, watch } from 'vue'

/**
 * 列表「整页刷新 + 滚动到底自动追加」通用逻辑。
 * 广场/日常记录/想法灵感/速记语录/读书记录等列表页共用。
 *
 * @param {Function} fetcher (params) => Promise<{ items, total }> 列表请求，params 含 page/limit
 * @param {{ pageSize?: number, nearBottom?: number }} options
 */
export function useInfiniteScroll(fetcher, { pageSize = 24, nearBottom = 260 } = {}) {
  const items = ref([])
  const total = ref(0)
  const loading = ref(false) // 正在请求（首屏 / 整页刷新 / 追加加载）
  const loaded = ref(false) // 至少成功完成过一次加载（用于空态判断）
  const sentinel = ref(null)

  let page = 1
  let scrollEl = null
  // 组件是否已卸载：卸载后到达的响应直接丢弃，避免给已销毁的组件赋值。
  let disposed = false
  // 进行中的请求序号。整页刷新时序号自增，旧响应回来发现序号过期就不写状态。
  let seq = 0

  // 找到最外层可滚动祖先（Element Plus 的 el-main），无则监听 window
  function findScrollParent(el) {
    while (el && el !== document.body) {
      const s = window.getComputedStyle(el)
      if (s.overflowY === 'auto' || s.overflowY === 'scroll') return el
      el = el.parentElement
    }
    return window
  }

  // reset=true 整页刷新（清空并从第 1 页加载）；false 追加下一页
  async function load(reset = true) {
    if (disposed) return
    // 整页刷新时即使有请求在飞也要能打断它（否则删完/存完再刷新会被 loading 挡住）
    if (loading.value && !reset) return
    const mySeq = ++seq
    loading.value = true
    const targetPage = reset ? 1 : page + 1
    if (reset) {
      page = 1
      items.value = []
    }
    try {
      const data = await fetcher({ page: targetPage, limit: pageSize })
      if (disposed || mySeq !== seq) return // 已卸载或已被更新的请求取代
      const list = Array.isArray(data?.items) ? data.items : []
      items.value = reset ? list : items.value.concat(list)
      total.value = Number(data?.total) || items.value.length
      page = targetPage
      loaded.value = true
    } catch {
      /* 错误已由 axios 拦截器提示 */
    } finally {
      if (mySeq === seq) loading.value = false
    }
  }

  function loadMore() {
    return load(false)
  }

  function onNearBottom() {
    if (loading.value || total.value === 0 || items.value.length >= total.value) return
    const el = scrollEl === window ? document.documentElement : scrollEl
    if (el.scrollTop + el.clientHeight >= el.scrollHeight - nearBottom) loadMore()
  }

  // 给哨兵挂监听。
  //
  // 原先这里（total 变化后）会立即执行一次 onNearBottom()「兜底补足一屏」——结果是不论
  // 视口多高，首屏之后都会立刻再拉一页：卡片数翻倍、图片请求翻倍，视口内每张卡片
  // 的图片都抢带宽，主线程被串行解码压满（表现为切菜单卡顿）。
  // 现在只在「首屏卡片确实撑不满视口（滚动条不够高）」时补一页，其余交给真实滚动。
  watch(
    total,
    async () => {
      await nextTick()
      if (!sentinel.value || items.value.length >= total.value) return
      if (!scrollEl) {
        scrollEl = findScrollParent(sentinel.value)
        scrollEl.addEventListener('scroll', onNearBottom, { passive: true })
      }
      const el = scrollEl === window ? document.documentElement : scrollEl
      // 内容高度不足一屏 → 用户无法滚动触发加载，此时才补
      if (el.scrollHeight <= el.clientHeight + nearBottom) loadMore()
    },
    { flush: 'post' },
  )

  onBeforeUnmount(() => {
    disposed = true
    scrollEl?.removeEventListener('scroll', onNearBottom)
  })

  return { items, total, loading, loaded, sentinel, load, loadMore }
}
