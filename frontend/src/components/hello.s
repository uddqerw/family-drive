# hello_win.s - 适用于 Windows + MinGW
    .global _main
    .text

_main:
    # Windows 下使用系统调用或 C 库
    pushq   %rbp
    movq    %rsp, %rbp
    
    # 调用 MessageBox (需要链接 user32.lib)
    # 这里简单返回 0
    movq    , %rax
    popq    %rbp
    ret
