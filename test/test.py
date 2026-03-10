import uiautomator2 as u2

d = u2.connect('emulator-5554')
d.app_start('com.android.chrome', stop=True) # Start Bilibili

d.implicitly_wait(10.0)

d(text="在裝置上新增帳戶").click()
