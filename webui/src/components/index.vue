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
        <a-card title="Welcome to the DPKI-ACME Prototype!">
          <a-typography-paragraph>This is the web UI of the DPKI-ACME prototype. The navigation bar on the left lets you switch between pages for control validation, certificate application and certificate management. If you have any questions, contact the developers via the About page.</a-typography-paragraph>
          <a-typography-paragraph style="text-align: right">DPKI-ACME Prototype Developers</a-typography-paragraph>
        </a-card>
        <a-row :gutter="[16,16]" :style="{marginTop: '20px'}">
          <a-col :span="16">
            <a-card title="DPKI-ACME Client Configuration" style="height: 350px">
            <p><b>Blockchain node: </b>{{ result.configInfo.blockchainNode }}</p>
            <p><b>Blockchain group: </b>{{ result.configInfo.blockchainGroup }}</p>
            <p><b>Blockchain ID: </b>{{ result.configInfo.blockchainID}}</p>
            <p><b>Blockchain account: </b>{{ result.configInfo.blockchainWallet }}</p>
            <p><b>DPKI-ACME contract address: </b>{{ result.configInfo.smartContractAddress }}</p>
            </a-card>
          </a-col>
          <a-col :span="8">
            <a-card title="DPKI-ACME Status" style="height: 350px" v-bind:result>
              <a-statistic title="IDs with Control Rights" :value="result.configInfo.controlValidationTimes"></a-statistic>
              <a-statistic title="Issued Certificates" :value="result.configInfo.certNum"></a-statistic>
              <a-statistic title="Emergency-Revoked Certificates" :value="result.configInfo.revokedCertNum"></a-statistic>
              <a-statistic title="Verified Certificates" :value="result.configInfo.verifiedCertNum"></a-statistic>
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
      style="color: Black"
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
  <a-float-button tooltip="Changelog" :style="{right: '24px',}" @click="showDrawer"></a-float-button>
</template>

<script setup>
  import {SolutionOutlined,CheckCircleOutlined,FileProtectOutlined,FileAddOutlined,HomeOutlined,AuditOutlined,InfoCircleOutlined,MenuUnfoldOutlined, MenuFoldOutlined} from '@ant-design/icons-vue';
  import { ref, onMounted, reactive } from 'vue';
  import { message } from 'ant-design-vue';
  import updateLog from "@/components/updateLog.vue";
  import axios from 'axios';
  const selectedKeys = ref(['1']);
  const collapsed = ref(false);
  // Drawer
  const open = ref(false);
  const afterOpenChange = bool => {
    console.log('open', bool);
  };
  const showDrawer = () => {
    open.value = true;
  };
  // Object type
  const result = reactive({
    configInfo: {
      "blockchainNode": '',
      "blockchainGroup": '',
      "blockchainID": '',
      "blockchainWallet": '',
      "smartContractAddress": '',
      "controlValidationTimes": 0,
      "certNum": 0,
      "revokedCertNum": 0,
      "verifiedCertNum": 0
    },
  });
  const fetchConfigInfo = async () => {
    try {
      const response = await axios.get('http://localhost:8080/api/configInfo');
      result.configInfo = response.data;
    } catch (error) {
      message.error('Failed to fetch the configuration:', error)
    }
  };
  onMounted(()=>{
    fetchConfigInfo();
  });



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
