<script setup lang="ts">
defineProps<{ name: 'file' | 'refresh' | 'settings' | 'chevron' | 'copy' | 'close' | 'check' }>()
const paths = {
  file: 'M9 3H5v14h14V7H9V3Zm0 0 6 0 4 4M9 3v4h10M8 11h8M8 14h5',
  refresh: 'M18 8a7 7 0 1 0 0 7M18 3v5h-5',
  settings: 'M9 3h4l1 3 3 1 2 3-2 3-1 3-3 1H9l-1-3-3-1-2-3 2-3 3-1 1-3ZM12 8a3 3 0 1 0 0 6 3 3 0 0 0 0-6Z',
  chevron: 'm7 9 5 5 5-5',
  copy: 'M8 8h11v12H8ZM15 8V3H3v12h5',
  close: 'm6 6 12 12M18 6 6 18',
  check: 'm5 12 4 4L19 6',
}
</script>

<template>
  <svg class="icon" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5" stroke-linecap="round" stroke-linejoin="round" aria-hidden="true"><path :d="paths[name]" /></svg>
</template>
