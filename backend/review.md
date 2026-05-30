1. ReconstructTask, ReconstructUser の時に引数に値オブジェクトをとっているがわざわざ値オブジェクトを取るのではなくプリミティブ値を取ればいいと思います。そうすれば ReconstructUserID などの関数もいらない
2. event.go にUserDeactivatedイベントがあるのは変です。user.go に移動させたほうがいいのでは？
3. FindByID, FindByAssigneeID をテーブル JOIN を使用して1度のクエリで実行したほうが良いのでは？
