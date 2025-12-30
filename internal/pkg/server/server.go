package server

import (
	"context"
	"fmt"
	"iter"
	"log/slog"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"google.golang.org/adk/agent"
	"google.golang.org/adk/artifact"
	"google.golang.org/adk/memory"
	"google.golang.org/adk/session"
	"google.golang.org/genai"
)

// Server 封装 Gin HTTP 服务器
type Server struct {
	router         *gin.Engine
	agentLoader    agent.Loader
	sessionService session.Service
	port           int
}

// NewServer 创建一个新的 HTTP 服务器
func NewServer(agentLoader agent.Loader, port int) *Server {
	gin.SetMode(gin.ReleaseMode)
	router := gin.New()
	router.Use(gin.Logger(), gin.Recovery())

	// 添加 CORS 支持
	router.Use(func(c *gin.Context) {
		c.Writer.Header().Set("Access-Control-Allow-Origin", "*")
		c.Writer.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		c.Writer.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")

		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(http.StatusNoContent)
			return
		}

		c.Next()
	})

	server := &Server{
		router:         router,
		agentLoader:    agentLoader,
		sessionService: session.InMemoryService(),
		port:           port,
	}

	server.setupRoutes()

	return server
}

// setupRoutes 设置路由
func (s *Server) setupRoutes() {
	api := s.router.Group("/api/v1")
	{
		// POST /api/v1/agents/:agentId/sessions/:sessionId
		api.POST("/agents/:agentId/sessions/:sessionId", s.handleAgentSession)
	}

	// 健康检查端点
	s.router.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})
}

// ChatRequest 表示聊天请求
type ChatRequest struct {
	Message string `json:"message" binding:"required"`
}

// handleAgentSession 处理 Agent 会话请求
func (s *Server) handleAgentSession(c *gin.Context) {
	agentID := c.Param("agentId")
	sessionID := c.Param("sessionId")

	var req ChatRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request: " + err.Error()})
		return
	}

	slog.Info("Received chat request",
		"agentId", agentID,
		"sessionId", sessionID,
		"message", req.Message)

	// 加载 Agent
	targetAgent, err := s.agentLoader.LoadAgent(agentID)
	if err != nil {
		slog.Error("Failed to load agent", "agentId", agentID, "err", err)
		c.JSON(http.StatusNotFound, gin.H{"error": "agent not found: " + agentID})
		return
	}

	// 设置流式响应头
	c.Writer.Header().Set("Content-Type", "text/event-stream")
	c.Writer.Header().Set("Cache-Control", "no-cache")
	c.Writer.Header().Set("Connection", "keep-alive")

	ctx := context.Background()

	// 获取或创建会话
	sess, err := s.getOrCreateSession(ctx, sessionID, agentID, targetAgent)
	if err != nil {
		slog.Error("Failed to get or create session", "err", err)
		s.sendSSE(c, "error", gin.H{"error": "failed to create session"})
		return
	}

	// 创建用户消息内容
	userContent := genai.NewContentFromText(req.Message, "user")

	// 创建调用上下文并执行 Agent
	invCtx := newInvocationContext(ctx, targetAgent, sess, userContent)

	// 执行 Agent 并处理事件流
	eventSeq := targetAgent.Run(invCtx)
	s.processEventStream(c, eventSeq)
}

// processEventStream 处理事件流并发送 SSE
func (s *Server) processEventStream(c *gin.Context, eventSeq iter.Seq2[*session.Event, error]) {
	eventSeq(func(event *session.Event, err error) bool {
		if err != nil {
			slog.Error("Event error", "err", err)
			s.sendSSE(c, "error", gin.H{"error": err.Error()})
			return true
		}

		if event == nil {
			return false
		}

		// 提取事件中的文本内容
		if event.Content != nil && len(event.Content.Parts) > 0 {
			for _, part := range event.Content.Parts {
				if part.Text != "" {
					s.sendSSE(c, "data", gin.H{"content": part.Text})
				}
			}
		}

		return true
	})

	// 发送结束标记
	s.sendSSE(c, "done", gin.H{"status": "completed"})
}

// sendSSE 发送服务器发送事件
func (s *Server) sendSSE(c *gin.Context, event string, data interface{}) {
	c.SSEvent(event, data)
	c.Writer.Flush()
}

// getOrCreateSession 获取或创建会话
func (s *Server) getOrCreateSession(ctx context.Context, sessionID, agentID string, targetAgent agent.Agent) (session.Session, error) {
	// 尝试获取现有会话
	getReq := &session.GetRequest{
		SessionID: sessionID,
		AppName:   agentID,
		UserID:    "default-user",
	}

	getResp, err := s.sessionService.Get(ctx, getReq)
	if err == nil && getResp.Session != nil {
		return getResp.Session, nil
	}

	// 创建新会话
	createReq := &session.CreateRequest{
		SessionID: sessionID,
		AppName:   agentID,
		UserID:    "default-user",
	}

	createResp, err := s.sessionService.Create(ctx, createReq)
	if err != nil {
		return nil, fmt.Errorf("failed to create session: %w", err)
	}

	return createResp.Session, nil
}

// invocationContext 实现 agent.InvocationContext 接口
type invocationContext struct {
	context.Context
	agent        agent.Agent
	session      session.Session
	userContent  *genai.Content
	invocationID string
	ended        bool
}

// newInvocationContext 创建新的调用上下文
func newInvocationContext(ctx context.Context, ag agent.Agent, sess session.Session, userContent *genai.Content) agent.InvocationContext {
	return &invocationContext{
		Context:      ctx,
		agent:        ag,
		session:      sess,
		userContent:  userContent,
		invocationID: fmt.Sprintf("inv-%s", strings.Replace(sess.ID(), "-", "", -1)),
		ended:        false,
	}
}

func (ic *invocationContext) Agent() agent.Agent {
	return ic.agent
}

func (ic *invocationContext) Artifacts() agent.Artifacts {
	// 返回空的 Artifacts 实现
	return &emptyArtifacts{}
}

func (ic *invocationContext) Memory() agent.Memory {
	// 返回空的 Memory 实现
	return &emptyMemory{}
}

func (ic *invocationContext) Session() session.Session {
	return ic.session
}

func (ic *invocationContext) InvocationID() string {
	return ic.invocationID
}

func (ic *invocationContext) Branch() string {
	return ic.agent.Name()
}

func (ic *invocationContext) UserContent() *genai.Content {
	return ic.userContent
}

func (ic *invocationContext) RunConfig() *agent.RunConfig {
	return &agent.RunConfig{}
}

func (ic *invocationContext) EndInvocation() {
	ic.ended = true
}

func (ic *invocationContext) Ended() bool {
	return ic.ended
}

// emptyArtifacts 提供一个空的 Artifacts 实现
type emptyArtifacts struct{}

func (ea *emptyArtifacts) Save(ctx context.Context, name string, data *genai.Part) (*artifact.SaveResponse, error) {
	return nil, fmt.Errorf("artifacts not supported")
}

func (ea *emptyArtifacts) Load(ctx context.Context, name string) (*artifact.LoadResponse, error) {
	return nil, fmt.Errorf("artifacts not supported")
}

func (ea *emptyArtifacts) LoadVersion(ctx context.Context, name string, version int) (*artifact.LoadResponse, error) {
	return nil, fmt.Errorf("artifacts not supported")
}

func (ea *emptyArtifacts) List(ctx context.Context) (*artifact.ListResponse, error) {
	return nil, fmt.Errorf("artifacts not supported")
}

// emptyMemory 提供一个空的 Memory 实现
type emptyMemory struct{}

func (em *emptyMemory) AddSession(ctx context.Context, sess session.Session) error {
	return fmt.Errorf("memory not supported")
}

func (em *emptyMemory) Search(ctx context.Context, query string) (*memory.SearchResponse, error) {
	return nil, fmt.Errorf("memory not supported")
}

// Start 启动 HTTP 服务器
func (s *Server) Start() error {
	addr := fmt.Sprintf(":%d", s.port)
	slog.Info("Starting HTTP server", "address", addr)
	return s.router.Run(addr)
}
