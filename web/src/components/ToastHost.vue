<script setup lang="ts">
import { useUiStore } from '@/stores/ui'

const ui = useUiStore()
</script>

<template>
  <div class="toast-host">
    <transition-group name="toast">
      <div v-for="t in ui.toasts" :key="t.id" class="toast" :class="t.kind">
        {{ t.text }}
      </div>
    </transition-group>
  </div>
</template>

<style scoped>
.toast-host {
  position: fixed;
  top: 18px;
  left: 50%;
  transform: translateX(-50%);
  z-index: 200;
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 8px;
  pointer-events: none;
}

.toast {
  padding: 9px 16px;
  border-radius: 9px;
  background: rgba(30, 30, 30, 0.9);
  color: #fff;
  font-size: 13px;
  max-width: 86vw;
  box-shadow: 0 6px 20px rgba(0, 0, 0, 0.18);
}

.toast.error {
  background: rgba(214, 60, 60, 0.95);
}

.toast-enter-active,
.toast-leave-active {
  transition: all 0.2s ease;
}

.toast-enter-from,
.toast-leave-to {
  opacity: 0;
  transform: translateY(-8px);
}
</style>
