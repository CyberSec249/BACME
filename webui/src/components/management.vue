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
          <a-typography-title style="text-align: center; margin-bottom: 20px">证书维护</a-typography-title>
        </a-typography>
        <a-card title="证书操作进度" :bordered="false" style="margin-top: 15px">
          <a-progress :percent="percent" :status="status" />
        </a-card>
        <a-card title="已申请证书的信息" style="margin-top: 15px">
          <a-table :columns="columns" :data-source="data" :bordered="true" size="small" :pagination="pagination" @change="handleTableChange">
            <template #bodyCell="{ column, record }">
              <template v-if="column.key === 'action'">
                <span>
                  <a-button style="margin-right: 5px" @click="updateCertificate(record)">Update</a-button>
                  <a-button style="margin-right: 5px" @click="revokeCertificate(record)">Revoke</a-button>
                </span>
              </template>
            </template>
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
  <a-float-button tooltip="更新日志" :style="{right: '24px',}" @click="showDrawer"></a-float-button>
</template>

<script setup>
import {SolutionOutlined,
  FileProtectOutlined,
  FileAddOutlined,
  HomeOutlined,
  AuditOutlined,
  InfoCircleOutlined,
  MenuUnfoldOutlined,
  MenuFoldOutlined,
  SmileOutlined, CheckCircleOutlined
} from '@ant-design/icons-vue';
  import {h, onMounted, ref} from 'vue';
  import updateLog from "@/components/updateLog.vue";
  import axios from "axios";
  import {message, notification} from "ant-design-vue";
  const selectedKeys = ref(['4']);
  const collapsed = ref(false);

  // 抽屉
  const open = ref(false);
  const afterOpenChange = bool => {
    console.log('open', bool);
  };
  const showDrawer = () => {
    open.value = true;
  };

  const pagination = ref({
    current: 1,
    pageSize: 10,
    total: 0,
    showSizeChanger: true,
    pageSizeOptions: ['10', '20', '50'],
    showTotal: (total) => `共 ${total} 条`,
  });

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
      width: '220px',
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
    {
      title: '操作',
      key: 'action',
    }
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

  // 定义更新表单数据
  const form1 = ref({
    ids: '',
    certHash: '',
  });
  // 定义撤销表单数据
  const form2 = ref({
    certHash: '',
  });
  // 更新请求
  const updateCertificate = async (record) => {
    try {
      startProgress()
      console.log(record);
      form1.value.ids = record.IDs;
      form1.value.certHash = record.certHash
      message.info("证书更新请求已提交")
      const response = await axios.post('http://localhost:8080/api/updateCertificate',form1.value);
      // 显示成功消息
      percent.value = 100;
      status.value = "";
      isRunning.value = false;
      openNotification("证书更新通知",response.data.message);
      await fetchCertInfo();
    } catch (error) {
      // 显示错误消息
      clearInterval(timer);
      status.value = "exception"; // 进度条失败
      isRunning.value = false;
      message.error('证书更新失败：' + error.message);
    }
  };
  // 撤销请求
  const revokeCertificate = async (record) => {
    try {
      startProgress()
      console.log(record);
      form2.value.certHash = record.certHash
      message.info("证书撤销请求已提交")
      const response = await axios.post('http://localhost:8080/api/revokeCertificate',form2.value);
      // 显示成功消息
      percent.value = 100;
      status.value = "";
      isRunning.value = false;
      openNotification("证书撤销通知",response.data.message);
      await fetchCertInfo();
    } catch (error) {
      // 显示错误消息
      clearInterval(timer);
      status.value = "exception"; // 进度条失败
      isRunning.value = false;
      message.error('证书撤销失败：' + error.message);
    }
  };
  // 表格分页或排序变化时触发
  const handleTableChange = (pag) => {
    pagination.value.current = pag.current;
    pagination.value.pageSize = pag.pageSize;
    fetchCertInfo();
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