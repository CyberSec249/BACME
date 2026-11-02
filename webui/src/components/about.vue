<!--Author: A1andNS-->
<template>
  <a-layout>
    <a-layout-sider v-model:collapsed="collapsed" :trigger="null" collapsible>
      <div class="logo">
        <img src="../assets/logo.png" alt="logo走丢了" style="width: 100%;height:100%" />
      </div>
      <a-menu v-model:selectedKeys="selectedKeys" theme="dark" mode="inline">
        <a-menu-item key="1">
          <HomeOutlined />
          <span><router-link to="/">概览</router-link></span>
        </a-menu-item>
        <a-menu-item key="2">
          <FileProtectOutlined  />
          <span><router-link to="/controlValidation">控制权验证</router-link></span>
        </a-menu-item>
        <a-menu-item key="3">
          <FileAddOutlined />
          <span><router-link to="/apply">证书申请</router-link></span>
        </a-menu-item>
        <a-menu-item key="4">
          <AuditOutlined />
          <span><router-link to="/management">证书维护</router-link></span>
        </a-menu-item>
        <a-menu-item key="5">
          <SolutionOutlined />
          <span><router-link to="/emergencyOperations">应急操作</router-link></span>
        </a-menu-item>
        <a-menu-item key="6">
          <CheckCircleOutlined />
          <span><router-link to="/verify">证书验证</router-link></span>
        </a-menu-item>
        <a-menu-item key="7">
          <InfoCircleOutlined />
          <span><router-link to="/about">关于系统</router-link></span>
        </a-menu-item>
      </a-menu>
    </a-layout-sider>
    <!-- 右边内容区域 -->
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
        <a-typography-title style="text-align: center; margin-bottom: 50px">DPKI-ACME原型系统WEBUI</a-typography-title>
        <a-typography-title :level="4" style="text-align: center; margin-bottom: 30px">本系统前端基于Vue3 + Ant Design Vue开发</a-typography-title >
        </a-typography>
        <a-row :gutter="[16,16]">
          <a-col :span="12">
            <a-card title="WEBUI版本" :bordered="true" style="text-align: center; height: 180px">
              <p>v0.3rc</p>
            </a-card>
          </a-col>
          <a-col :span="12">
            <a-card title="联系作者" :bordered="true" style="text-align: center; height: 180px">
              <p>E-mail: <a href="mailto:lilongxinalan@live.com">lilongxinalan@live.com</a></p>
              <p>Github: <a href="https://github.com/A1andNS">https://github.com/A1andNS</a></p>
            </a-card>
          </a-col>
          <a-col :span="12">
            <a-card title="致谢" :bordered="true" style="text-align: center; height: 190px">
              <p>华为技术有限公司</p>
              <p>西安电子科技大学网络与信息安全学院信任管理创新团队</p>
              <p>西安电子科技大学广州研究院</p>
            </a-card>
          </a-col>
          <a-col :span="12">
            <a-card title="拾遗" :bordered="true" style="text-align: center; height: 190px">
              <p>莫愁千里路</p>
              <p>自有到来风</p>
            </a-card>
          </a-col>
        </a-row>
      </a-layout-content>
<!--   内容底部   -->
      <a-layout-footer style="text-align: center">
        DPKI-ACME原型系统 ©2024 Created by A1andNS 西安电子科技大学
      </a-layout-footer>
    </a-layout>
  </a-layout>
<!--  抽屉-->
  <a-drawer
      v-model:open="open"
      class="custom-class"
      root-class-name="root-class-name"
      :root-style="{ color: 'blue' }"
      style="color: Black; background: ghostwhite"
      title="更新日志"
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
<!--  浮动按钮-->
  <a-float-button tooltip="更新日志" :style="{right: '24px'}" @click="showDrawer"></a-float-button>
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
  // 抽屉
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