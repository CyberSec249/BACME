<!--Author: A1andNS-->
<template>
  <a-layout>
    <a-layout-sider v-model:collapsed="collapsed" :trigger="null" collapsible>
      <div class="logo">
        <img src="../assets/logo.png" alt="logo not found" style="width: 100%;height:100%" />
      </div>
      <a-menu v-model:selectedKeys="selectedKeys" theme="dark" mode="inline">
        <a-menu-item key="1">
          <HomeOutlined />
          <span><router-link to="/">Overview</router-link></span>
        </a-menu-item>
        <a-menu-item key="2">
          <FileProtectOutlined  />
          <span><router-link to="/controlValidation">Control Validation</router-link></span>
        </a-menu-item>
        <a-menu-item key="3">
          <FileAddOutlined />
          <span><router-link to="/apply">Certificate Application</router-link></span>
        </a-menu-item>
        <a-menu-item key="4">
          <AuditOutlined />
          <span><router-link to="/management">Certificate Management</router-link></span>
        </a-menu-item>
        <a-menu-item key="5">
          <SolutionOutlined />
          <span><router-link to="/emergencyOperations">Emergency Operations</router-link></span>
        </a-menu-item>
        <a-menu-item key="6">
          <CheckCircleOutlined />
          <span><router-link to="/verify">Certificate Verification</router-link></span>
        </a-menu-item>
        <a-menu-item key="7">
          <InfoCircleOutlined />
          <span><router-link to="/about">About</router-link></span>
        </a-menu-item>
      </a-menu>
    </a-layout-sider>
    <!-- Content area on the right -->
    <a-layout>
      <a-layout-header style="background: #fff; padding: 0">
        <menu-unfold-outlined
            v-if="collapsed"
            class="trigger"
            @click="() => (collapsed = !collapsed)"
        />
        <menu-fold-outlined v-else class="trigger" @click="() => (collapsed = !collapsed)" />
      </a-layout-header>
      <a-layout-content
          :style="{ margin: '24px 16px', padding: '24px', minHeight: '640px' }"
      >
        <a-typography>
        <a-typography-title style="text-align: center; margin-bottom: 50px">DPKI-ACME Prototype Web UI</a-typography-title>
        <a-typography-title :level="4" style="text-align: center; margin-bottom: 30px">The frontend is built with Vue 3 + Ant Design Vue</a-typography-title >
        </a-typography>
        <a-row :gutter="[16,16]">
          <a-col :span="12">
            <a-card title="Web UI Version" :bordered="true" style="text-align: center; height: 180px">
              <p>v0.3rc</p>
            </a-card>
          </a-col>
          <a-col :span="12">
            <a-card title="Contact" :bordered="true" style="text-align: center; height: 180px">
              <p>E-mail: <a href="mailto:lilongxinalan@live.com">lilongxinalan@live.com</a></p>
              <p>Github: <a href="https://github.com/A1andNS">https://github.com/A1andNS</a></p>
            </a-card>
          </a-col>
          <a-col :span="12">
            <a-card title="Acknowledgments" :bordered="true" style="text-align: center; height: 190px">
              <p>Huawei Technologies Co., Ltd.</p>
              <p>Trust Management Innovation Team, School of Cyber Engineering, Xidian University</p>
              <p>Guangzhou Institute of Technology, Xidian University</p>
            </a-card>
          </a-col>
          <a-col :span="12">
            <a-card title="Miscellany" :bordered="true" style="text-align: center; height: 190px">
              <p>Worry not about the long road ahead;</p>
              <p>a fair wind will come in its own time.</p>
            </a-card>
          </a-col>
        </a-row>
      </a-layout-content>
<!--   Footer   -->
      <a-layout-footer style="text-align: center">
        DPKI-ACME Prototype ©2024 Created by A1andNS, Xidian University
      </a-layout-footer>
    </a-layout>
  </a-layout>
<!--  Drawer-->
  <a-drawer
      v-model:open="open"
      class="custom-class"
      root-class-name="root-class-name"
      :root-style="{ color: 'blue' }"
      style="color: Black; background: ghostwhite"
      title="Changelog"
      placement="right"
      @after-open-change="afterOpenChange"
  >
    <a-row :gutter="[16,16]">
      <a-col :span="24" v-for="log in updateLog.data().updateLogs">
        <a-card v-bind:title=log.title :bordered="true">
          <p>{{log.content}}</p>
        </a-card>
      </a-col>
    </a-row>
  </a-drawer>
<!--  Floating button-->
  <a-float-button tooltip="Changelog" :style="{right: '24px'}" @click="showDrawer"></a-float-button>
</template>

<script setup>
import {SolutionOutlined,
  FileProtectOutlined,
  FileAddOutlined,
  HomeOutlined,
  AuditOutlined,
  InfoCircleOutlined,
  MenuUnfoldOutlined,
  MenuFoldOutlined, CheckCircleOutlined
} from '@ant-design/icons-vue';
  import {ref} from 'vue';
  import updateLog from './updateLog.vue'

  const selectedKeys = ref(['7']);
  const collapsed = ref(false);
  // Drawer
  const open = ref(false);
  const afterOpenChange = bool => {
    console.log('open', bool);
  };
  const showDrawer = () => {
    open.value = true;
  };

</script>

<style>
.trigger {
  font-size: 18px;
  line-height: 64px;
  padding: 0 24px;
  cursor: pointer;
  transition: color 0.3s;
}

.trigger:hover {
  color: #1890ff;
}

.logo {
  height: 32px;
  background: rgba(255, 255, 255, 0.3);
  margin: 16px;
}
</style>