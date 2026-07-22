<template>
  <el-result icon="warning" title="页面不存在" sub-title="当前地址没有对应的管理页面，请从左侧菜单重新选择。">
    <template #extra>
      <el-button type="primary" @click="router.replace('/dashboard')">返回控制台</el-button>
    </template>
  </el-result>
</template>

<script setup lang="ts">
import { useRouter } from 'vue-router'

const router = useRouter()
</script>
