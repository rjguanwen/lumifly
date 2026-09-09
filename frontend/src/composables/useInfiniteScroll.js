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
    if (loading.value) return
    loading.value = true
    const targetPage = reset ? 1 : page + 1
    if (reset) {
      page = 1
      items.value = []
    }
    try {
      const data = await fetcher({ page: targetPage, limit: pageSize })
      const list = Array.isArray(data?.items) ? data.items : []
      items.value = reset ? list : items.value.concat(list)
      total.value = Number(data?.total) || items.value.length
      page = targetPage
      loaded.value = true
    } catch {
      /* 错误已由 axios 拦截器提示 */
    } finally {
      loading.value = false
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

  // 每次 total 变化后（首屏返回、整页刷新返回）给哨兵挂上滚动监听，并兜底补足一屏
  watch(
    total,
    async () => {
      await nextTick()
      if (!sentinel.value || items.value.length >= total.value) return
      if (!scrollEl) {
        scrollEl = findScrollParent(sentinel.value)
        scrollEl.addEventListener('scroll', onNearBottom, { passive: true })
      }
      onNearBottom()
    },
    { flush: 'post' },
  )

  onBeforeUnmount(() => scrollEl?.removeEventListener('scroll', onNearBottom))

  return { items, total, loading, loaded, sentinel, load, loadMore }
}
