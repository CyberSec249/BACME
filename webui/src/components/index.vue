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
        <a-card title="欢迎使用DPKI-ACME原型系统！">
          <a-typography-paragraph>这里是DPKI-ACME原型系统WEBUI界面，欢迎您的使用！左侧导航栏提供了不同操作页面的便捷切换，您可以轻松切换到控制权验证，证书申请以及后续证书维护模块，如有疑问可以在关于页面联系开发者！</a-typography-paragraph>
          <a-typography-paragraph style="text-align: right">DPKI-ACME原型系统开发者</a-typography-paragraph>
        </a-card>
        <a-row :gutter="[16,16]" :style="{marginTop: '20px'}">
          <a-col :span="16">
            <a-card title="DPKI-ACME客户端配置" style="height: 350px">
            <p><b>区块链连接节点：</b>{{ result.configInfo.blockchainNode }}</p>
            <p><b>区块链群组号：</b>{{ result.configInfo.blockchainGroup }}</p>
            <p><b>区块链ID：</b>{{ result.configInfo.blockchainID}}</p>
            <p><b>区块链钱包账户：</b>{{ result.configInfo.blockchainWallet }}</p>
            <p><b>DPKI-ACME智能合约地址：</b>{{ result.configInfo.smartContractAddress }}</p>
            </a-card>
          </a-col>
          <a-col :span="8">
            <a-card title="DPKI-ACME工作情况" style="height: 350px" v-bind:result>
              <a-statistic title="具有控制权的ID数量" :value="result.configInfo.controlValidationTimes"></a-statistic>
              <a-statistic title="已申请证书数量" :value="result.configInfo.certNum"></a-statistic>
              <a-statistic title="已紧急撤销证书数量" :value="result.configInfo.revokedCertNum"></a-statistic>
              <a-statistic title="已验证证书数量" :value="result.configInfo.verifiedCertNum"></a-statistic>
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
      style="color: Black"
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
  <a-float-button tooltip="更新日志" :style="{right: '24px',}" @click="showDrawer"></a-float-button>
</template>

<script setup>
  import {SolutionOutlined,CheckCircleOutlined,FileProtectOutlined,FileAddOutlined,HomeOutlined,AuditOutlined,InfoCircleOutlined,MenuUnfoldOutlined, MenuFoldOutlined} from '@ant-design/icons-vue';
  import { ref, onMounted, reactive } from 'vue';
  import { message } from 'ant-design-vue';
  import updateLog from "@/components/updateLog.vue";
  import axios from 'axios';
  const selectedKeys = ref(['1']);
  const collapsed = ref(false);
  // 抽屉
  const open = ref(false);
  const afterOpenChange = bool => {
    console.log('open', bool);
  };
  const showDrawer = () => {
    open.value = true;
  };
  // 对象类型
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
      message.error('无法获取匹配的配置信息:', error)
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
