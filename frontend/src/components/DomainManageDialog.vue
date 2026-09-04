<template>
  <el-dialog :model-value="visible" title="书籍领域管理" width="520px" @update:model-value="emit('update:visible', $event)">
    <!-- 添加区 -->
    <div class="flex gap-2 mb-3">
      <el-input v-model="newName" placeholder="输入新领域名称，如：悬疑推理" @keyup.enter="addDomain" clearable />
      <el-button type="primary" :loading="adding" @click="addDomain">添加</el-button>
    </div>
    <el-button size="small" text type="primary" :loading="addingPreset" @click="addPresets">
      一键添加常用领域
    </el-button>

    <el-divider class="!my-3" />

    <!-- 领域列表 -->
    <div class="space-y-2 max-h-[46vh] overflow-y-auto">
      <div
        v-for="d in domains"
        :key="d.id"
        class="domain-row flex items-center gap-2 p-2 rounded-lg hover:bg-gray-50 border border-gray-100"
      >
        <el-icon color="#9ca3af"><Collection /></el-icon>
        <el-input v-model="d.editName" size="small" class="flex-1" @keyup.enter="saveName(d)" />
        <el-button size="small" type="primary" text :loading="d.saving === 'rename'" @click="saveName(d)">
          <el-icon><Check /></el-icon>
        </el-button>
        <el-button size="small" type="danger" text :loading="d.saving === 'del'" @click="removeDomain(d)">
          <el-icon><Delete /></el-icon>
        </el-button>
      </div>
      <el-empty v-if="!domains.length" description="还没有领域，先添加一个吧" :image-size="60" />
    </div>
  </el-dialog>
</template>

<script setup>
import { reactive, ref, watch } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import { bookDomainApi, defaultBookDomains } from '../api'

const props = defineProps({
  visible: { type: Boolean, default: false },
})
const emit = defineEmits(['update:visible', 'changed'])

const domains = ref([])
const newName = ref('')
const adding = ref(false)
const addingPreset = ref(false)

async function load() {
  try {
    const list = (await bookDomainApi.list()) || []
    domains.value = list.map((d) => ({ ...d, editName: d.name, saving: null }))
  } catch { /* 忽略 */ }
}

watch(
  () => props.visible,
  (v) => v && load(),
)

async function addDomain() {
  const name = newName.value.trim()
  if (!name) {
    ElMessage.warning('请输入领域名称')
    return
  }
  adding.value = true
  try {
    await bookDomainApi.create(name)
    newName.value = ''
    emit('changed')
    await load()
  } catch { /* 拦截器已提示 */ } finally {
    adding.value = false
  }
}

async function addPresets() {
  addingPreset.value = true
  const existing = new Set(domains.value.map((d) => d.name))
  try {
    for (const name of defaultBookDomains) {
      if (!existing.has(name)) await bookDomainApi.create(name)
    }
    ElMessage.success('常用领域已添加')
    emit('changed')
    await load()
  } catch { /* 忽略 */ } finally {
    addingPreset.value = false
  }
}

async function saveName(d) {
  const name = d.editName.trim()
  if (!name) {
    ElMessage.warning('名称不能为空')
    return
  }
  if (name === d.name) return
  d.saving = 'rename'
  try {
    await bookDomainApi.rename(d.id, name)
    ElMessage.success('已重命名')
    emit('changed')
    await load()
  } catch { /* 拦截器已提示 */ } finally {
    d.saving = null
  }
}

async function removeDomain(d) {
  try {
    await ElMessageBox.confirm(
      `删除领域「${d.name}」？使用该领域的书籍将清空领域。`,
      '提示',
      { type: 'warning' },
    )
  } catch {
    return
  }
  d.saving = 'del'
  try {
    await bookDomainApi.remove(d.id)
    emit('changed')
    await load()
  } catch { /* 拦截器已提示 */ } finally {
    d.saving = null
  }
}
</script>
