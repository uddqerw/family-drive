import pyautogui
import time
import threading
from PIL import ImageGrab
import numpy as np
import tkinter as tk
from tkinter import ttk, messagebox

class AdvancedGreenClicker:
    def __init__(self):
        self.monitor_region = None
        self.click_position = None
        self.green_threshold = 200
        self.is_monitoring = False
        self.thread = None
        
    def select_monitor_region(self):
        """选择监控区域"""
        messagebox.showinfo("选择监控区域", 
                          "请在5秒内将鼠标移动到监控区域的左上角")
        time.sleep(5)
        x1, y1 = pyautogui.position()
        
        messagebox.showinfo("选择监控区域", 
                          "请在5秒内将鼠标移动到监控区域的右下角")
        time.sleep(5)
        x2, y2 = pyautogui.position()
        
        self.monitor_region = (min(x1, x2), min(y1, y2), max(x1, x2), max(y1, y2))
        return self.monitor_region
    
    def select_click_position(self):
        """选择点击位置"""
        messagebox.showinfo("选择点击位置", 
                          "请在5秒内将鼠标移动到要点击的位置")
        time.sleep(5)
        self.click_position = pyautogui.position()
        return self.click_position
    
    def check_green_region(self):
        """检查区域是否变绿"""
        if not self.monitor_region:
            return False
            
        screenshot = ImageGrab.grab(bbox=self.monitor_region)
        screenshot_array = np.array(screenshot)
        
        if screenshot_array.shape[2] == 4:
            screenshot_array = screenshot_array[:, :, :3]
        
        green_channel = screenshot_array[:, :, 1]
        avg_green = np.mean(green_channel)
        
        return avg_green > self.green_threshold
    
    def start_monitoring_thread(self):
        """在单独线程中监控"""
        def monitor():
            print("开始监控...")
            while self.is_monitoring:
                if self.check_green_region():
                    print("检测到绿色区域，执行点击！")
                    self.perform_click()
                    break
                time.sleep(0.1)
        
        self.thread = threading.Thread(target=monitor)
        self.thread.daemon = True
        self.thread.start()
    
    def perform_click(self):
        """执行点击操作"""
        if self.click_position:
            pyautogui.click(self.click_position[0], self.click_position[1])
            print(f"已在位置 {self.click_position} 点击")
    
    def start(self):
        """开始监控"""
        if not self.monitor_region or not self.click_position:
            messagebox.showerror("错误", "请先选择监控区域和点击位置")
            return
        
        self.is_monitoring = True
        self.start_monitoring_thread()
        print("监控已启动")
    
    def stop(self):
        """停止监控"""
        self.is_monitoring = False
        print("监控已停止")

# GUI界面
class GreenClickerGUI:
    def __init__(self):
        self.clicker = AdvancedGreenClicker()
        self.root = tk.Tk()
        self.root.title("绿色屏幕点击器")
        self.setup_ui()
    
    def setup_ui(self):
        # 主框架
        main_frame = ttk.Frame(self.root, padding="10")
        main_frame.grid(row=0, column=0, sticky=(tk.W, tk.E, tk.N, tk.S))
        
        # 监控区域选择
        ttk.Button(main_frame, text="选择监控区域", 
                  command=self.select_region).grid(row=0, column=0, pady=5, sticky=tk.W)
        self.region_label = ttk.Label(main_frame, text="未选择")
        self.region_label.grid(row=0, column=1, pady=5)
        
        # 点击位置选择
        ttk.Button(main_frame, text="选择点击位置", 
                  command=self.select_click).grid(row=1, column=0, pady=5, sticky=tk.W)
        self.click_label = ttk.Label(main_frame, text="未选择")
        self.click_label.grid(row=1, column=1, pady=5)
        
        # 阈值设置
        ttk.Label(main_frame, text="绿色阈值:").grid(row=2, column=0, pady=5, sticky=tk.W)
        self.threshold_var = tk.StringVar(value="200")
        ttk.Entry(main_frame, textvariable=self.threshold_var, width=10).grid(row=2, column=1, pady=5)
        
        # 控制按钮
        ttk.Button(main_frame, text="开始监控", 
                  command=self.start_monitoring).grid(row=3, column=0, pady=10)
        ttk.Button(main_frame, text="停止监控", 
                  command=self.stop_monitoring).grid(row=3, column=1, pady=10)
        
        # 状态显示
        self.status_label = ttk.Label(main_frame, text="就绪", foreground="green")
        self.status_label.grid(row=4, column=0, columnspan=2, pady=5)
    
    def select_region(self):
        region = self.clicker.select_monitor_region()
        self.region_label.config(text=f"区域: {region}")
    
    def select_click(self):
        position = self.clicker.select_click_position()
        self.click_label.config(text=f"位置: {position}")
    
    def start_monitoring(self):
        try:
            self.clicker.green_threshold = int(self.threshold_var.get())
            self.clicker.start()
            self.status_label.config(text="监控中...", foreground="blue")
        except ValueError:
            messagebox.showerror("错误", "请输入有效的阈值数字")
    
    def stop_monitoring(self):
        self.clicker.stop()
        self.status_label.config(text="已停止", foreground="red")
    
    def run(self):
        self.root.mainloop()

# 使用示例
if __name__ == "__main__":
    # 简单版本使用
    print("=== 简单版本 ===")
    # monitor_region = (0, 0, 100, 100)  # 根据实际情况修改
    # click_position = (500, 500)        # 根据实际情况修改
    # clicker = GreenScreenClicker(monitor_region, click_position)
    # clicker.start_monitoring()
    
    # GUI版本使用
    print("=== GUI版本 ===")
    app = GreenClickerGUI()
    app.run()