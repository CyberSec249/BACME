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
          <SolutionOutlined />
          <span><router-link to="/apply">证书申请</router-link></span>
        </a-menu-item>
        <a-menu-item key="4">
          <AuditOutlined />
          <span><router-link to="/management">证书维护</router-link></span>
        </a-menu-item>
        <a-menu-item key="5">
          <AuditOutlined />
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
          <a-typography-title style="text-align: center; margin-bottom: 20px">证书申请</a-typography-title>
        </a-typography>
        <a-card title="证书申请表单" :bordered="false">
          <a-form :model="form" @submit="handleSubmit">
            <a-form-item label="ID" :label-col="{ span: 4 }" :wrapper-col="{ span: 16 }">
              <a-input v-model:value="form.ids" placeholder="请输入需要进行证书申请的所有ID" />
              <p>说明：ID的格式为一个或多个ID使用逗号分隔。例如：www.example1.com 或 www.example1.com,www.example2.com,127.0.0.1</p>
            </a-form-item>
            <a-form-item :wrapper-col="{ offset: 4, span: 16 }">
              <a-button type="primary" html-type="submit">提交</a-button>
            </a-form-item>
          </a-form>
        </a-card>
        <a-card title="证书申请进度" :bordered="false" style="margin-top: 15px">
          <a-progress :percent="percent" :status="status" />
        </a-card>
        <a-card title="已申请证书的信息" style="margin-top: 15px">
          <a-table :columns="columns" :data-source="data" :bordered="true" size="small">
          </a-table>
        </a-card>
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
  <a-float-button tooltip="更新日志" :style="{right: '24px'}" @click="showDrawer"></a-float-button>
</template>

<script setup>
import {
  FileProtectOutlined,
  SolutionOutlined,
  HomeOutlined,
  AuditOutlined,
  InfoCircleOutlined,
  MenuUnfoldOutlined,
  MenuFoldOutlined,
  SmileOutlined, CheckCircleOutlined
} from '@ant-design/icons-vue';
import {h, onMounted, ref} from 'vue';
  import updateLog from "@/components/updateLog.vue";
  import {message, notification} from "ant-design-vue";
  import axios from "axios";
  const selectedKeys = ref(['3']);
  const collapsed = ref(false);

  // 抽屉
  const open = ref(false);
  const afterOpenChange = bool => {
    console.log('open', bool);
  };
  const showDrawer = () => {
    open.value = true;
  };

  // 通知
  const openNotification = (title,content) => {
    notification.open({
      message: title,
      description:
      content,
      icon: () =>
          h(SmileOutlined, {
            style: 'color: #108ee9',
          })
    });
  };

  // 进度条
  const percent = ref(0); // 进度百分比
  const isRunning = ref(false); // 是否正在运行
  const status = ref("active"); //动态
  let timer = null; // 定时器
  const startProgress = () => {
    if (isRunning.value) return; // 如果已经在运行，则退出
    isRunning.value = true;
    percent.value = 0; //重置为0

    timer = setInterval(() => {
      if (percent.value < 90) {
        percent.value += 30; // 每次增加 30%
      }
      else if (percent.value >= 90 && percent.value < 99) {
        percent.value += 3;
      }
      else {
        clearInterval(timer); // 达到 99% 后停止,等待后端指令
      }
    }, 300); // 每 300 毫秒更新一次
  };

  // 定义表单数据
  const form = ref({
    ids: '',
  });

  // 提交表单
  const handleSubmit = async (e) => {
    e.preventDefault(); // 阻止默认表单提交行为
    try {
      // 发送 POST 请求到后端
      message.info("证书请求已提交")
      startProgress()
      const response = await axios.post('http://localhost:8080/api/applyCertificate', form.value);
      // 显示成功消息
      percent.value = 100;
      status.value = "";
      isRunning.value = false;
      openNotification("证书申请通知",response.data.message);
      await fetchCertInfo();
    } catch (error) {
      // 显示错误消息
      clearInterval(timer);
      status.value = "exception"; // 进度条失败
      isRunning.value = false;
      message.error('证书提交失败：' + error.message);
    }
  };

  // 对象类型
  const data = ref([]);

  const columns = [
    {
      title: '证书Hash值',
      dataIndex: 'certHash',
      key: 'certHash',
    },
    {
      title: '标识符',
      dataIndex: 'IDs',
      key: 'IDs',
      ellipsis: true
    },
    {
      title: '公钥哈希',
      dataIndex: 'PKHash',
      key: 'PKHash',
    },
    {
      title: '有效期',
      dataIndex: 'validTime',
      key: 'validTime',
    },
  ];
  // 获取table数据
  const fetchCertInfo = async () => {
    try {
      const response = await axios.get('http://localhost:8080/api/getCertInfo');
      data.value = response.data;
    } catch (error) {
      message.error('无法获取证书信息:', error)
    }
  };
  onMounted(()=>{
    fetchCertInfo();
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

.site-layout {
  background: #fff;
}
</style>