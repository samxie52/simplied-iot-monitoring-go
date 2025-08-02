// 简单的Chart.js测试文件
console.log('Chart.js测试文件加载成功');

// 创建一个简单的Chart构造函数用于测试
if (typeof window !== 'undefined') {
    window.Chart = function(ctx, config) {
        console.log('Chart构造函数被调用:', ctx, config);
        this.ctx = ctx;
        this.config = config;
        this.data = config.data || {};
        this.options = config.options || {};
        
        // 模拟Chart.js的基本方法
        this.update = function() {
            console.log('Chart.update() 被调用');
        };
        
        this.destroy = function() {
            console.log('Chart.destroy() 被调用');
        };
        
        this.resize = function() {
            console.log('Chart.resize() 被调用');
        };
        
        console.log('Chart实例创建成功');
        return this;
    };
    
    // 添加版本信息
    window.Chart.version = '4.4.0-test';
    
    console.log('Chart.js测试版本已设置:', window.Chart.version);
}
