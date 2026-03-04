return {
  "atiladefreitas/dooing",
  config = function()
    require("dooing").setup({
      window = {
        width = 100, -- Width of the floating window
        height = 30, -- Height of the floating window
        border = "rounded", -- Border style: 'single', 'double', 'rounded', 'solid'
        position = "center", -- Window position: 'right', 'left', 'top', 'bottom', 'center',
      },
    })
  end,
}
