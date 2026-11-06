<template>
  <!-- 顶部细进度条：路由切换期间显示，让"点击已被接收"这件事可见。
       目的不是修性能，而是避免用户在等待时以为点击丢了、反复点击，
       反而加重主线程与请求的负担。 -->
  <transition name="rp-fade">
    <div v-if="active" class="rp-bar" role="progressbar" aria-label="页面加载中">
      <i class="rp-fill" />
    </div>
  </transition>
</template>

<script setup>
import { onBeforeUnmount, onMounted, ref } from 'vue'
import { useRouter } from 'vue-router'

const router = useRouter()
const active = ref(false)
let startTimer = null
let minTimer = null
const MIN_VISIBLE = 220 // 极快的切换不闪一下，保持至少 220ms

function start() {
  if (minTimer) {
    clearTimeout(minTimer)
    minTimer = null
  }
  // 延迟 80ms 再显示：本地瞬时跳转不出现进度条，避免闪烁
  if (startTimer) clearTimeout(startTimer)
  startTimer = setTimeout(() => {
    active.value = true
  }, 80)
}

function done() {
  if (startTimer) {
    clearTimeout(startTimer)
    startTimer = null
  }
  if (!active.value) return
  minTimer = setTimeout(() => {
    active.value = false
    minTimer = null
  }, MIN_VISIBLE)
}

onMounted(() => {
  router.beforeEach(() => {
    start()
    return true
  })
  router.afterEach(() => done())
  router.onError(() => done())
})

onBeforeUnmount(() => {
  if (startTimer) clearTimeout(startTimer)
  if (minTimer) clearTimeout(minTimer)
})
</script>

<style scoped>
.rp-bar {
  position: fixed;
  top: 0;
  left: 0;
  right: 0;
  height: 2px;
  z-index: 3000;
  overflow: hidden;
  background: rgba(91, 124, 255, 0.12);
}

.rp-fill {
  display: block;
  height: 100%;
  width: 40%;
  border-radius: 2px;
  background: linear-gradient(90deg, rgba(91, 124, 255, 0), #5b7cff 40%, #7c9bff);
  animation: rp-run 1s ease-in-out infinite;
}

@keyframes rp-run {
  0% {
    transform: translateX(-100%);
  }
  100% {
    transform: translateX(250%);
  }
}

.rp-fade-enter-active,
.rp-fade-leave-active {
  transition: opacity 0.18s ease;
}

.rp-fade-enter-from,
.rp-fade-leave-to {
  opacity: 0;
}
</style>
