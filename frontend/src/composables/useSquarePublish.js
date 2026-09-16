import { ref, watch } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import { squareApi } from '../api'

/**
 * 列表页「发布到广场」状态与操作（一次拉取我发布的全部内容，本地按类型映射，避免每条请求）。
 *
 * @param {string} sourceType 内容类型：record / milestone / idea / book / quick
 * @param {import('vue').Ref<Array>} items 列表数据（变化时自动刷新发布状态）
 * @param {{ label?: string }} options label 用于提示文案
 */
export function useSquarePublish(sourceType, items, { label = '内容' } = {}) {
  // id -> { id: publicationId, status }
  const pubMap = ref({})
  const publishingId = ref(null)

  async function loadPubState() {
    try {
      const data = await squareApi.mine()
      const map = {}
      for (const p of data.items || []) {
        if (p.sourceType === sourceType) {
          map[Number(p.sourceId)] = { id: p.id, status: p.status || 'published' }
        }
      }
      pubMap.value = map
    } catch {
      /* 忽略 */
    }
  }

  async function publish(item) {
    publishingId.value = item.id
    try {
      const p = await squareApi.publish(sourceType, item.id)
      const status = p.status || 'published'
      pubMap.value[item.id] = { id: p.id, status }
      if (status === 'pending') {
        ElMessage.info('内容可能包含敏感信息，已转入人工审核，通过后将在广场展示')
      } else if (status === 'rejected') {
        ElMessage.warning('该内容未能通过审核，请修改后重新提交')
      } else {
        ElMessage.success('已发布到广场')
      }
    } catch {
      /* 拦截器已提示 */
    } finally {
      publishingId.value = null
    }
  }

  async function unpublish(item) {
    const rec = pubMap.value[item.id]
    if (!rec) return
    try {
      await ElMessageBox.confirm(`撤回后这条${label}将不再在广场展示，确定撤回？`, '撤回发布', { type: 'warning' })
    } catch {
      return
    }
    publishingId.value = item.id
    try {
      await squareApi.unpublish(rec.id)
      delete pubMap.value[item.id]
      ElMessage.success('已从广场撤回')
    } catch {
      /* 拦截器已提示 */
    } finally {
      publishingId.value = null
    }
  }

  // 审核中：可撤回后修改重发，或继续等待
  async function pending(item) {
    const rec = pubMap.value[item.id]
    if (!rec) return
    try {
      await ElMessageBox.confirm(
        `该${label}正在人工审核中。如需修改后重新发布，请先撤回；不操作则继续等待审核结果。`,
        '内容审核中',
        { type: 'info', confirmButtonText: '撤回', cancelButtonText: '继续等待' },
      )
    } catch {
      return
    }
    publishingId.value = item.id
    try {
      await squareApi.unpublish(rec.id)
      delete pubMap.value[item.id]
      ElMessage.success('已撤回，修改后重新点击发布即可')
    } catch {
      /* 拦截器已提示 */
    } finally {
      publishingId.value = null
    }
  }

  // 列表（含滚动追加）变化后刷新发布状态
  if (items) watch(items, () => { loadPubState() })

  return { pubMap, publishingId, loadPubState, publish, unpublish, pending }
}
